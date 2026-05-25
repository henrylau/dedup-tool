// Package service implements the Wails-bound API for folder similarity.
package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"folder-similarity/app/convert"
	"folder-similarity/app/dto"
	"folder-similarity/app/pathutil"
	"folder-similarity/core"
	"folder-similarity/internal/videothumb"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Similarity is the Wails service backing the tree/compare UI.
type Similarity struct {
	mu sync.RWMutex

	phase  string
	errMsg string

	rootAbs    string
	storage    *core.MemoryStorage
	checker    *core.SimilarityChecker
	selected   string
	shutdown   context.Context
	scanCancel context.CancelFunc

	// Similarity groups for the current tree selection (GetSimilarityFolderGroup).
	similarityGroups [][2]*core.FolderSimilarity
	groupIndex       int   // 0..n-1, or -1 if GroupCount>1 and user has not chosen yet
	groupLabels      []string
	// merge is the active comparison (GenerateMergeFolderPair) when groupIndex >= 0.
	merge   core.MergeFolderPair
	mergeOK bool
	mergeF1 *core.FolderSimilarity
	mergeF2 *core.FolderSimilarity

	applyRunning bool
	applyCancel  context.CancelFunc
}

// ServiceName implements application.ServiceName.
func (s *Similarity) ServiceName() string {
	return "Similarity"
}

// ServiceShutdown cancels a running scan.
func (s *Similarity) ServiceShutdown() error {
	s.cancelScan()
	return nil
}

func (s *Similarity) cancelScan() {
	s.mu.Lock()
	c := s.scanCancel
	s.scanCancel = nil
	s.mu.Unlock()
	if c != nil {
		c()
	}
}

func (s *Similarity) emitLog(msg string) {
	app := application.Get()
	if app == nil {
		return
	}
	app.Event.Emit("log", msg)
}

func (s *Similarity) emitPhase(ev dto.PhaseEvent) {
	app := application.Get()
	if app == nil {
		return
	}
	app.Event.Emit("phase", ev)
}

func (s *Similarity) emitProgress(ev dto.ProgressEvent) {
	app := application.Get()
	if app == nil {
		return
	}
	app.Event.Emit("progress", ev)
}

func (s *Similarity) emitScanProgress(ev dto.ScanProgressEvent) {
	app := application.Get()
	if app == nil {
		return
	}
	app.Event.Emit("scanProgress", ev)
}

func (s *Similarity) clearMergeStateLocked() {
	s.similarityGroups = nil
	s.groupIndex = 0
	s.groupLabels = nil
	s.merge = core.MergeFolderPair{}
	s.mergeOK = false
	s.mergeF1, s.mergeF2 = nil, nil
}

func (s *Similarity) compareRowCount() int {
	if !s.mergeOK {
		return 0
	}
	return len(s.merge.FolderPairs) + len(s.merge.FilePairs)
}

// GetState returns a snapshot of lifecycle, selection, and group-picker state.
func (s *Similarity) GetState() dto.AppState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	gc := len(s.similarityGroups)
	need := gc > 1 && s.groupIndex < 0
	return dto.AppState{
		Phase:     s.phase,
		Message:   s.errMsg,
		RootPath:  s.rootAbs,
		HasData:   s.storage != nil && s.checker != nil,
		Selected:  s.selected,
		Scanning:  s.phase == dto.PhaseScanning,
		GroupCount:    gc,
		GroupIndex:    s.groupIndex,
		GroupLabels:   append([]string(nil), s.groupLabels...),
		NeedGroupPick: need,
		CompareRowCount: s.compareRowCount(),
		ApplyRunning:    s.applyRunning,
	}
}

// PickRootFolder opens a directory picker. Returns the absolute path or an empty string if cancelled.
func (s *Similarity) PickRootFolder() (string, error) {
	app := application.Get()
	if app == nil {
		return "", fmt.Errorf("application not initialised")
	}
	path, err := app.Dialog.OpenFile().CanChooseFiles(false).CanChooseDirectories(true).
		SetTitle("Select folder to scan").PromptForSingleSelection()
	if err != nil {
		return "", err
	}
	return path, nil
}

// PickJSONFile opens a file picker for JSON (saved db).
func (s *Similarity) PickJSONFile() (string, error) {
	app := application.Get()
	if app == nil {
		return "", fmt.Errorf("application not initialised")
	}
	path, err := app.Dialog.OpenFile().CanChooseFiles(true).CanChooseDirectories(false).
		AddFilter("JSON", "*.json").SetTitle("Select JSON data file").PromptForSingleSelection()
	if err != nil {
		return "", err
	}
	return path, nil
}

// LoadFromJSONFile loads a previously exported file list.
func (s *Similarity) LoadFromJSONFile(abs string) error {
	if s.applyRunningNow() {
		return fmt.Errorf("apply in progress; cancel or wait before loading data")
	}
	abs = filepath.Clean(abs)
	if abs == "" || abs == "." {
		return fmt.Errorf("invalid path")
	}
	if _, err := os.Stat(abs); err != nil {
		return err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return err
	}
	s.emitPhase(dto.PhaseEvent{Phase: dto.PhaseLoading, Message: "Loading JSON…"})
	s.emitLog("Loading JSON: " + abs)

	s.mu.Lock()
	s.errMsg = ""
	s.phase = dto.PhaseLoading
	s.rootAbs = ""
	s.cancelScan()
	s.storage = core.NewMemoryStorage()
	s.checker = &core.SimilarityChecker{}
	s.selected = "."
	s.clearMergeStateLocked()

	var files []*core.File
	if err = json.Unmarshal(data, &files); err != nil {
		s.errMsg = err.Error()
		s.phase = dto.PhaseError
		s.mu.Unlock()
		s.emitLog("Error: " + err.Error())
		s.emitPhase(dto.PhaseEvent{Phase: dto.PhaseError, Message: err.Error()})
		return err
	}
	for _, file := range files {
		if file == nil {
			continue
		}
		if err = s.storage.AddFile(file); err != nil {
			s.errMsg = err.Error()
			s.phase = dto.PhaseError
			s.mu.Unlock()
			s.emitLog("Error: " + err.Error())
			s.emitPhase(dto.PhaseEvent{Phase: dto.PhaseError, Message: err.Error()})
			return err
		}
	}
	if err = s.checker.CalculateSimilarity(s.storage); err != nil {
		s.errMsg = err.Error()
		s.phase = dto.PhaseError
		s.mu.Unlock()
		s.emitLog("Error: " + err.Error())
		s.emitPhase(dto.PhaseEvent{Phase: dto.PhaseError, Message: err.Error()})
		return err
	}
	s.phase = dto.PhaseReady
	s.mu.Unlock()
	s.emitLog("Load complete, similarity ready.")
	s.emitPhase(dto.PhaseEvent{Phase: dto.PhaseReady, Message: "ready"})
	return nil
}

// StartScan sets root to an absolute path and scans asynchronously.
func (s *Similarity) StartScan(rootAbs string) error {
	if s.applyRunningNow() {
		return fmt.Errorf("apply in progress; cancel or wait before scanning")
	}
	rootAbs, err := filepath.Abs(filepath.Clean(rootAbs))
	if err != nil {
		return err
	}
	st, err := os.Stat(rootAbs)
	if err != nil {
		return err
	}
	if !st.IsDir() {
		return fmt.Errorf("not a directory: %s", rootAbs)
	}

	s.cancelScan()
	scanCtx, cancel := context.WithCancel(context.Background())

	s.mu.Lock()
	s.rootAbs = rootAbs
	s.storage = core.NewMemoryStorage()
	s.checker = &core.SimilarityChecker{}
	s.errMsg = ""
	s.selected = "."
	s.clearMergeStateLocked()
	s.phase = dto.PhaseScanning
	s.scanCancel = cancel
	s.mu.Unlock()

	s.emitPhase(dto.PhaseEvent{Phase: dto.PhaseScanning, Message: "scanning"})
	s.emitLog("Scan started: " + rootAbs)

	go s.runScan(scanCtx, rootAbs)
	return nil
}

func (s *Similarity) runScan(ctx context.Context, root string) {
	s.mu.RLock()
	st := s.storage
	ch := s.checker
	s.mu.RUnlock()
	if st == nil || ch == nil {
		s.setError("internal: storage not initialised for scan")
		return
	}

	var scanTotal, lastEmit int
	emitScanCount := func(n int) {
		scanTotal = n
		if n == 1 || n-lastEmit >= 50 {
			s.emitScanProgress(dto.ScanProgressEvent{FilesScanned: n})
			lastEmit = n
		}
	}

	sc := core.Scanner{
		Storage: st,
		Path:    []string{root},
		Context: ctx,
		Logger: func(msg string) {
			s.emitLog(msg)
		},
		OnFileAdded: emitScanCount,
	}
	err := sc.Scan()
	if ctx.Err() == nil && scanTotal > lastEmit {
		s.emitScanProgress(dto.ScanProgressEvent{FilesScanned: scanTotal})
	}
	if err != nil {
		if ctx.Err() != nil {
			s.mu.Lock()
			s.phase = dto.PhaseIdle
			s.scanCancel = nil
			s.storage = nil
			s.checker = nil
			s.rootAbs = ""
			s.selected = "."
			s.clearMergeStateLocked()
			s.errMsg = ""
			s.mu.Unlock()
			s.emitLog("Scan cancelled")
			s.emitPhase(dto.PhaseEvent{Phase: dto.PhaseIdle, Message: "cancelled"})
		} else {
			s.setError(err.Error())
		}
		return
	}

	if err := ch.CalculateSimilarity(st); err != nil {
		s.setError(err.Error())
		return
	}

	s.mu.Lock()
	s.phase = dto.PhaseReady
	s.scanCancel = nil
	s.mu.Unlock()
	s.emitLog("Scan complete, similarity ready.")
	s.emitPhase(dto.PhaseEvent{Phase: dto.PhaseReady, Message: "ready"})
}

func (s *Similarity) setError(msg string) {
	s.mu.Lock()
	s.errMsg = msg
	s.phase = dto.PhaseError
	s.scanCancel = nil
	s.mu.Unlock()
	s.emitLog("Error: " + msg)
	s.emitPhase(dto.PhaseEvent{Phase: dto.PhaseError, Message: msg})
}

// CancelScan aborts a running directory scan.
func (s *Similarity) CancelScan() {
	s.cancelScan()
}

// GetTree returns the folder tree, optionally pruned to folders that participate in similarity.
func (s *Similarity) GetTree(similarityOnly bool) (dto.FolderNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.storage == nil || s.checker == nil {
		return dto.FolderNode{}, fmt.Errorf("no data: scan or load JSON first")
	}
	root, err := s.storage.GetFolder(".")
	if err != nil {
		return dto.FolderNode{}, err
	}
	if !similarityOnly {
		return convert.TreeFromFolder(root), nil
	}
	keep := func(p string) bool { return s.checker.ContainsSimilarityGroup(p) }
	return convert.TreeFromFolderPrune(root, keep), nil
}

func groupLabel(g [2]*core.FolderSimilarity) string {
	if g[0] == nil || g[1] == nil {
		return ""
	}
	t := g[1].Path
	f1, f2 := g[0], g[1]
	return fmt.Sprintf("%s (F1: %d/%d %.1f%% | F2: %d/%d %.1f%%)", t,
		f1.DuplicateFileCount, f1.FileCount, f1.DuplicatedPercentage(),
		f2.DuplicateFileCount, f2.FileCount, f2.DuplicatedPercentage())
}

// SelectFolder sets the selected folder and either builds the merge (one group) or opens group selection (many).
func (s *Similarity) SelectFolder(rel string) error {
	rel = pathutil.NormalizeRelStorageKey(strings.TrimSpace(rel))
	if rel == "" {
		rel = "."
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.storage == nil || s.checker == nil {
		return fmt.Errorf("no data")
	}
	if rel != "." && (strings.HasPrefix(rel, string(filepath.Separator)) || strings.Contains(rel, "..")) {
		return fmt.Errorf("invalid folder key")
	}
	if s.rootAbs != "" {
		abs := rel
		if rel != "." {
			abs = filepath.Join(s.rootAbs, rel)
		} else {
			abs = s.rootAbs
		}
		abs, _ = filepath.Abs(abs)
		if !pathutil.IsRootOrUnder(s.rootAbs, abs) {
			return fmt.Errorf("path outside scan root")
		}
	}

	if _, err := s.storage.GetFolder(rel); err != nil {
		return err
	}
	s.selected = rel

	groups := s.checker.GetSimilarityFolderGroup(rel)
	s.similarityGroups = groups
	s.groupLabels = nil
	for _, g := range groups {
		s.groupLabels = append(s.groupLabels, groupLabel(g))
	}

	if len(groups) == 0 {
		s.clearMergeStateLocked()
		return nil
	}
	if len(groups) == 1 {
		s.groupIndex = 0
		s.buildMergeLocked(0)
		return nil
	}
	// Multiple groups: wait for SelectSimilarityGroupIndex.
	s.groupIndex = -1
	s.merge = core.MergeFolderPair{}
	s.mergeOK = false
	s.mergeF1, s.mergeF2 = nil, nil
	return nil
}

func (s *Similarity) buildMergeLocked(i int) {
	g := s.similarityGroups[i]
	s.mergeF1, s.mergeF2 = g[0], g[1]
	s.merge = s.checker.GenerateMergeFolderPair(g[0], g[1])
	s.groupIndex = i
	s.mergeOK = true
}

// SelectSimilarityGroupIndex applies when multiple groups exist; index is 0-based.
func (s *Similarity) SelectSimilarityGroupIndex(i int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.similarityGroups) < 2 {
		return fmt.Errorf("group selection not needed")
	}
	if i < 0 || i >= len(s.similarityGroups) {
		return fmt.Errorf("group index out of range")
	}
	s.buildMergeLocked(i)
	return nil
}

func parseMergeActionCode(s string) (core.MergeAction, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "none", "":
		return core.ActionNone, nil
	case "deleteright", "delete-right":
		return core.ActionDeleteRight, nil
	case "deleteleft", "delete-left":
		return core.ActionDeleteLeft, nil
	case "movetoright", "move-to-right", "moveright":
		return core.ActionMoveToRight, nil
	case "movetoleft", "move-to-left", "moveleft":
		return core.ActionMoveToLeft, nil
	default:
		return core.ActionNone, fmt.Errorf("unknown action %q", s)
	}
}

func (s *Similarity) setRowAction(index int, a core.MergeAction) {
	nF := len(s.merge.FolderPairs)
	if index < nF {
		s.merge.FolderPairs[index].SetAction(a)
		return
	}
	s.merge.FilePairs[index-nF].SetAction(a)
}

// SetCompareRowAction sets the merge action for a row (folder rows first, then file rows, 0-based).
func (s *Similarity) SetCompareRowAction(rowIndex int, action string) error {
	act, err := parseMergeActionCode(action)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.mergeOK {
		return fmt.Errorf("no active merge")
	}
	n := len(s.merge.FolderPairs) + len(s.merge.FilePairs)
	if rowIndex < 0 || rowIndex >= n {
		return fmt.Errorf("row index out of range")
	}
	s.setRowAction(rowIndex, act)
	return nil
}

// ClearCompareRowAction clears a single row (same as action "none").
func (s *Similarity) ClearCompareRowAction(rowIndex int) error {
	return s.SetCompareRowAction(rowIndex, "none")
}

// SetAllCompareRowActions sets the same action on every row.
func (s *Similarity) SetAllCompareRowActions(action string) error {
	act, err := parseMergeActionCode(action)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.mergeOK {
		return fmt.Errorf("no active merge")
	}
	for i := range s.merge.FolderPairs {
		s.merge.FolderPairs[i].SetAction(act)
	}
	for i := range s.merge.FilePairs {
		s.merge.FilePairs[i].SetAction(act)
	}
	return nil
}

// ClearAllCompareRowActions sets every row to none.
func (s *Similarity) ClearAllCompareRowActions() error {
	return s.SetAllCompareRowActions("none")
}

// collectActionTasks matches ui/comparelist GetActions (file rows first, then folder rows, then delete-empty).
func collectActionTasks(merge *core.MergeFolderPair, f1, f2 *core.FolderSimilarity) []core.FileActionTask {
	actions := []core.FileActionTask{}
	for i := range merge.FilePairs {
		pair := &merge.FilePairs[i]
		if pair.Action != core.ActionNone {
			actions = append(actions, pair.GetActionTask(f1, f2))
		}
	}
	for i := range merge.FolderPairs {
		pair := &merge.FolderPairs[i]
		if pair.Action != core.ActionNone {
			actions = append(actions, pair.GetActionTask(f1, f2)...)
		}
	}
	if f1 != nil {
		actions = append(actions, core.FileActionTask{Action: core.DeleteEmptyFolder, Folder: f1.Folder})
	}
	if f2 != nil {
		actions = append(actions, core.FileActionTask{Action: core.DeleteEmptyFolder, Folder: f2.Folder})
	}
	return actions
}

// GetApplyPreview returns human-readable action lines (phase 4 dry run; does not execute).
func (s *Similarity) GetApplyPreview() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.mergeOK {
		return nil, fmt.Errorf("no active merge: select a folder and group first")
	}
	tasks := collectActionTasks(&s.merge, s.mergeF1, s.mergeF2)
	out := make([]string, 0, len(tasks))
	for _, t := range tasks {
		if line := t.String(); line != "" {
			out = append(out, line)
		}
	}
	return out, nil
}

// GetMergePreview returns the compare view including per-row action state.
func (s *Similarity) GetMergePreview() (dto.MergePreview, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.storage == nil {
		return dto.MergePreview{}, fmt.Errorf("no data")
	}
	if s.selected == "" {
		return dto.MergePreview{}, fmt.Errorf("no folder selected")
	}
	if len(s.similarityGroups) == 0 {
		return convert.BuildInteractiveMergePreview(nil, "No similarity group for this folder.", 0, 0, false), nil
	}
	if len(s.similarityGroups) > 1 && s.groupIndex < 0 {
		return convert.BuildInteractiveMergePreview(
			nil, "Select a similarity group to compare (see toolbar or list).", len(s.similarityGroups), -1, true), nil
	}
	if !s.mergeOK {
		return dto.MergePreview{}, fmt.Errorf("no merge to display")
	}
	note := ""
	if len(s.similarityGroups) > 1 {
		note = fmt.Sprintf("Group %d of %d (sorted).", s.groupIndex+1, len(s.similarityGroups))
	}
	return convert.BuildInteractiveMergePreview(&s.merge, note, len(s.similarityGroups), s.groupIndex, true), nil
}

const maxGalleryImageBytes = 15 << 20 // cap RAM / IPC for raw image reads

const maxGalleryVideoBytes = 512 << 20 // video thumbs stream-decode; size gate limits abuse only

var galleryImageExts = map[string]string{
	"jpg":  "image/jpeg",
	"jpeg": "image/jpeg",
	"png":  "image/png",
	"gif":  "image/gif",
	"webp": "image/webp",
	"bmp":  "image/bmp",
	"svg":  "image/svg+xml",
	"ico":  "image/x-icon",
}

// galleryVideoExts aligns with explorer gallery “video” kind (astiavthumbnail path).
var galleryVideoExts = map[string]struct{}{
	"mp4": {}, "mov": {}, "mkv": {}, "avi": {}, "webm": {}, "m4v": {},
}

var galleryHEIFExts = map[string]struct{}{
	"heic": {}, "heif": {},
}

// ReadGalleryImagePreview reads an image file under the current scan root and returns a data URL
// for display in the webview (raw file:// URLs are often blocked by embedded browsers).
//
// HEIF (.heic / .heif) are transcoded with go-astiav where CGO + FFmpeg/heif codecs are present.
//
// Supported video extensions return a synthesized JPEG thumbnail (first frame ~1s, downscaled via go-astiav).
func (s *Similarity) ReadGalleryImagePreview(relPath string) (string, error) {
	relPath = strings.TrimSpace(relPath)
	if relPath == "" {
		return "", fmt.Errorf("empty path")
	}

	s.mu.RLock()
	root := s.rootAbs
	s.mu.RUnlock()
	if root == "" {
		return "", fmt.Errorf("no scan root")
	}

	rel := pathutil.NormalizeRelStorageKey(relPath)
	if rel == "." || rel == "" {
		return "", fmt.Errorf("invalid path")
	}
	if strings.HasPrefix(rel, string(filepath.Separator)) || strings.Contains(rel, "..") {
		return "", fmt.Errorf("invalid path")
	}

	abs := filepath.Join(root, rel)
	var err error
	abs, err = filepath.Abs(abs)
	if err != nil {
		return "", err
	}
	if !pathutil.IsRootOrUnder(root, abs) {
		return "", fmt.Errorf("path outside scan root")
	}

	st, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !st.Mode().IsRegular() {
		return "", fmt.Errorf("not a regular file")
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(abs), "."))

	if _, ok := galleryVideoExts[ext]; ok {
		if st.Size() > maxGalleryVideoBytes {
			return "", fmt.Errorf("video too large for preview")
		}
		jpegBytes, err := videothumb.EncodeJPEGThumbnail(abs, videothumb.Options{
			SeekSeconds: 1,
			MaxWidth:    480,
			Quality:     80,
			DebugLog: func(msg string) {
				s.emitLog("[video-thumb] " + msg)
			},
		})
		if err != nil {
			s.emitLog("[video-thumb] failed: " + err.Error())
			return "", fmt.Errorf("video thumbnail: %w", err)
		}
		b64 := base64.StdEncoding.EncodeToString(jpegBytes)
		return fmt.Sprintf("data:image/jpeg;base64,%s", b64), nil
	}

	if _, ok := galleryHEIFExts[ext]; ok {
		if st.Size() > maxGalleryVideoBytes {
			return "", fmt.Errorf("HEIF image too large for preview")
		}
		jpegBytes, err := videothumb.EncodeJPEGThumbnail(abs, videothumb.Options{
			NoSeek:   true,
			MaxWidth: 480,
			Quality:  85,
			DebugLog: func(msg string) {
				s.emitLog("[heif-preview] " + msg)
			},
		})
		if err != nil {
			s.emitLog("[heif-preview] failed: " + err.Error())
			return "", fmt.Errorf("HEIF preview: %w", err)
		}
		b64 := base64.StdEncoding.EncodeToString(jpegBytes)
		return fmt.Sprintf("data:image/jpeg;base64,%s", b64), nil
	}

	if st.Size() > maxGalleryImageBytes {
		return "", fmt.Errorf("file too large for preview")
	}

	if _, ok := galleryImageExts[ext]; !ok {
		return "", fmt.Errorf("unsupported image type")
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		return "", err
	}

	mt := mime.TypeByExtension(filepath.Ext(abs))
	if mt == "" || !strings.HasPrefix(mt, "image/") {
		mt = galleryImageExts[ext]
	}
	if mt == "" || !strings.HasPrefix(mt, "image/") {
		return "", fmt.Errorf("not an image")
	}

	b64 := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", mt, b64), nil
}

// HomeDir returns the current user's home directory. Used by the server-mode folder picker
// to seed its initial location.
func (s *Similarity) HomeDir() (string, error) {
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Clean(h), nil
}

// ListDir lists immediate child directories of path. If filterExt is non-empty (e.g. ".json"),
// matching files are included as well. Used by the server-mode folder/file picker.
// path must be absolute.
func (s *Similarity) ListDir(path string, filterExt string) ([]dto.DirEntry, error) {
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("path must be absolute")
	}
	abs := filepath.Clean(path)
	entries, err := os.ReadDir(abs)
	if err != nil {
		return nil, err
	}
	filterExt = strings.ToLower(filterExt)
	out := make([]dto.DirEntry, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		isDir := e.IsDir()
		if !isDir {
			if filterExt == "" {
				continue
			}
			if strings.ToLower(filepath.Ext(name)) != filterExt {
				continue
			}
		}
		out = append(out, dto.DirEntry{
			Name:  name,
			Path:  filepath.Join(abs, name),
			IsDir: isDir,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsDir != out[j].IsDir {
			return out[i].IsDir
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}
