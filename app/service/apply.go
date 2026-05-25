package service

import (
	"context"
	"errors"
	"fmt"
	"folder-similarity/app/dto"
	"folder-similarity/core"
)

// applyLogger adapts the core executor to the Wails log stream.
type applyLogger struct{ emit func(string) }

func (a *applyLogger) Info(m string)  { a.emit(m) }
func (a *applyLogger) Error(m string) { a.emit("Error: " + m) }

func (s *Similarity) applyRunningNow() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.applyRunning
}

func (s *Similarity) finishApply() {
	s.mu.Lock()
	s.applyRunning = false
	s.applyCancel = nil
	s.mu.Unlock()
}

func buildApplySummaryString(tasks []core.FileActionTask) string {
	moveCount, deleteCount, replaceCount, nonDuplicateDeleteCount, deleteFolderCount, moveFolderCount := 0, 0, 0, 0, 0, 0
	for _, action := range tasks {
		switch action.Action {
		case core.Move:
			if action.TargetName == "" {
				moveCount++
			} else {
				replaceCount++
			}
		case core.Delete:
			deleteCount++
			if action.NotDuplicate {
				nonDuplicateDeleteCount++
			}
		case core.DeleteFolder:
			deleteFolderCount++
		case core.MoveFolder:
			moveFolderCount++
		}
	}
	// Wording matches ui/mainmodel.go HandleApplyActions.
	return fmt.Sprintf(
		"Apply following actions:\nMove %d files, delete %d files, replace %d files\nDelete  %d  Non-duplicate files, delete %d folders, move %d folders",
		moveCount, deleteCount, replaceCount, nonDuplicateDeleteCount, deleteFolderCount, moveFolderCount,
	)
}

// GetApplySummary returns a user-facing confirmation string (TUI-style counts) plus root.
func (s *Similarity) GetApplySummary() (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.mergeOK {
		return "", fmt.Errorf("no active merge: select a folder and group first")
	}
	if s.rootAbs == "" {
		return "", fmt.Errorf("no scan root on disk: use Pick & scan before apply (JSON-only load has no real root)")
	}
	tasks := collectActionTasks(&s.merge, s.mergeF1, s.mergeF2)
	return fmt.Sprintf(
		"Root: %s\n\n%s\n\n(Delete-empty folder steps are still run at the end; they are not included in the counts above.)",
		s.rootAbs, buildApplySummaryString(tasks),
	), nil
}

// ApplyExecute runs the on-disk executor for the current merge. Errors during execution are sent on the "progress" event; this returns only validation / start-up errors.
func (s *Similarity) ApplyExecute() error {
	s.mu.Lock()
	if s.applyRunning {
		s.mu.Unlock()
		return fmt.Errorf("apply already in progress")
	}
	if !s.mergeOK {
		s.mu.Unlock()
		return fmt.Errorf("no active merge: select a folder and group first")
	}
	if s.rootAbs == "" {
		s.mu.Unlock()
		return fmt.Errorf("no scan root on disk: use Pick & scan before apply (JSON-only load has no real root)")
	}
	if s.storage == nil {
		s.mu.Unlock()
		return fmt.Errorf("no data")
	}
	if len(s.similarityGroups) > 1 && s.groupIndex < 0 {
		s.mu.Unlock()
		return fmt.Errorf("select a similarity group first")
	}
	tasks := collectActionTasks(&s.merge, s.mergeF1, s.mergeF2)
	st := s.storage
	root := s.rootAbs
	s.applyRunning = true
	ctx, cancel := context.WithCancel(s.shutdownContext())
	s.applyCancel = cancel
	s.mu.Unlock()

	go s.runApplyAsync(ctx, root, st, tasks)
	return nil
}

func (s *Similarity) shutdownContext() context.Context {
	if s.shutdown == nil {
		return context.Background()
	}
	return s.shutdown
}

// CancelApply requests cancellation of a running apply (same as TUI cancel during executor).
func (s *Similarity) CancelApply() {
	s.mu.Lock()
	c := s.applyCancel
	s.mu.Unlock()
	if c != nil {
		c()
	}
}

func (s *Similarity) runApplyAsync(ctx context.Context, root string, st core.Storage, tasks []core.FileActionTask) {
	defer s.finishApply()

	s.emitProgress(dto.ProgressEvent{Phase: "running", Message: "starting", Current: 0, Total: len(tasks)})

	log := &applyLogger{emit: s.emitLog}
	exec := core.NewExecutor(st, root, tasks, log)
	err := s.runExecutorWithProgress(ctx, exec)
	if errors.Is(err, context.Canceled) {
		s.emitProgress(dto.ProgressEvent{Phase: "cancelled", Message: "cancelled"})
		return
	}
	if err != nil {
		s.emitLog("Apply failed: " + err.Error())
		s.emitProgress(dto.ProgressEvent{Phase: "error", Error: err.Error(), Message: err.Error()})
		return
	}
	s.emitProgress(dto.ProgressEvent{Phase: "done", Message: "complete", Current: len(tasks), Total: len(tasks)})
	s.refreshAfterApply()
}

func (s *Similarity) runExecutorWithProgress(ctx context.Context, exec *core.Executor) error {
	errCh := make(chan error, 1)
	go func() { errCh <- exec.Execute(ctx) }()
	for {
		select {
		case p, ok := <-exec.ProgressChannel():
			if !ok {
				return <-errCh
			}
			s.emitProgress(dto.ProgressEvent{
				Phase:   "running",
				Current: p.Current,
				Total:   p.Total,
				Message: p.Message,
			})
		case err := <-errCh:
			s.drainProgress(exec)
			return err
		}
	}
}

func (s *Similarity) drainProgress(exec *core.Executor) {
	for {
		select {
		case p, ok := <-exec.ProgressChannel():
			if !ok {
				return
			}
			s.emitProgress(dto.ProgressEvent{
				Phase:   "running",
				Current: p.Current,
				Total:   p.Total,
				Message: p.Message,
			})
		default:
			return
		}
	}
}

func (s *Similarity) refreshAfterApply() {
	s.mu.RLock()
	sel := s.selected
	st := s.storage
	ch := s.checker
	s.mu.RUnlock()
	if st == nil || ch == nil {
		return
	}
	if err := ch.CalculateSimilarity(st); err != nil {
		s.emitLog("Refresh after apply: " + err.Error())
		return
	}
	if err := s.SelectFolder(sel); err != nil {
		s.emitLog("Re-select after apply: " + err.Error())
		if err2 := s.SelectFolder("."); err2 != nil {
			s.emitLog("Retry select root: " + err2.Error())
		}
	}
	s.emitLog("View refreshed after apply.")
}
