package core

import (
	"context"
	"errors"
	"os"
	"time"
)

// Executor handles execution of file action tasks with progress reporting.
type Executor struct {
	storage      Storage
	rootPath     string
	tasks        []FileActionTask
	logger       Logger
	done         bool
	progressChan chan ProgressUpdate
}

// ProgressUpdate represents a progress update during task execution.
type ProgressUpdate struct {
	Current int
	Total   int
	Message string
}

// NewExecutor creates a new executor instance.
func NewExecutor(storage Storage, rootPath string, tasks []FileActionTask, logger Logger) *Executor {
	return &Executor{
		storage:      storage,
		rootPath:     rootPath,
		tasks:        tasks,
		logger:       logger,
		progressChan: make(chan ProgressUpdate, 10),
	}
}

// ProgressChannel returns the progress update channel.
func (e *Executor) ProgressChannel() <-chan ProgressUpdate {
	return e.progressChan
}

// Execute runs all tasks with progress reporting and cancellation support.
func (e *Executor) Execute(ctx context.Context) error {
	root, err := os.OpenRoot(e.rootPath)
	if err != nil {
		return err
	}
	defer root.Close()

	totalTasks := len(e.tasks)
	for i, task := range e.tasks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// TODO: execute task
			err := ExecuteFileActionTask(e.storage, root, &task)
			message := task.String()

			if err != nil && errors.Is(err, ErrNotEmptyFolder) {
				message = message + " (folder is not empty)"
				err = nil
			}

			// Send progress update
			select {
			case e.progressChan <- ProgressUpdate{
				Current: i + 1,
				Total:   totalTasks,
				Message: message,
			}:
			default:
				// Channel is full, skip this update
			}

			// log result
			if e.logger != nil {
				if err != nil {
					e.logger.Error(err.Error())
				} else {
					e.logger.Info("Executed task: " + message)
				}
			}
			time.Sleep(10 * time.Millisecond)

			if err != nil {
				return err
			}
		}
	}
	e.done = true

	return nil
}
