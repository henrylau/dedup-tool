package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"folder-similarity/app/pathutil"
	"folder-similarity/core"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// ExportDataToJSON writes the current in-memory file list to a path chosen in a save dialog (phase 6; same data shape as the TUI’s ExportStorage / db.json).
func (s *Similarity) ExportDataToJSON() error {
	if s.applyRunningNow() {
		return fmt.Errorf("apply in progress; wait or cancel before export")
	}
	s.mu.RLock()
	st := s.storage
	s.mu.RUnlock()
	if st == nil {
		return fmt.Errorf("no data to export: scan or load JSON first")
	}
	data, err := st.ExportStorage()
	if err != nil {
		return err
	}
	app := application.Get()
	if app == nil {
		return fmt.Errorf("application not initialised")
	}
	path, err := app.Dialog.SaveFile().
		AddFilter("JSON", "*.json").
		SetFilename("db.json").
		SetMessage("Save exported file list as JSON").
		PromptForSingleSelection()
	if err != nil {
		return err
	}
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}
	s.emitLog("Exported file list to: " + path)
	return nil
}

// GetExportedJSON returns the current in-memory file list as a JSON string.
// Used by server-mode frontend to trigger a browser download (Blob + <a download>).
func (s *Similarity) GetExportedJSON() (string, error) {
	if s.applyRunningNow() {
		return "", fmt.Errorf("apply in progress; wait or cancel before export")
	}
	s.mu.RLock()
	st := s.storage
	s.mu.RUnlock()
	if st == nil {
		return "", fmt.Errorf("no data to export: scan or load JSON first")
	}
	data, err := st.ExportStorage()
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// RevealMergeFolder opens the system file manager for the left or right folder of the active merge (phase 6; TUI: OpenFileExplorer). side is "left" or "right".
func (s *Similarity) RevealMergeFolder(side string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.rootAbs == "" {
		return fmt.Errorf("no scan root: export/open in Finder only works with a real folder scan (not JSON-only load)")
	}
	if !s.mergeOK {
		return fmt.Errorf("no active merge: select a folder and group first")
	}
	var fs *core.FolderSimilarity
	switch strings.ToLower(strings.TrimSpace(side)) {
	case "left":
		fs = s.mergeF1
	case "right":
		fs = s.mergeF2
	default:
		return fmt.Errorf(`side must be "left" or "right"`)
	}
	if fs == nil {
		return fmt.Errorf("no folder on that side")
	}
	rel := fs.Folder.Path
	abs, err := s.absPathUnderRootUnlocked(rel)
	if err != nil {
		return err
	}
	return revealAbsInOS(abs)
}

func openAbsWithDefaultApp(abs string) error {
	abs = filepath.Clean(abs)
	if _, err := os.Stat(abs); err != nil {
		return err
	}
	switch runtime.GOOS {
	case "windows":
		// Let start handle files and folders (including paths with spaces when quoted inside argv).
		return exec.Command("cmd", "/c", "start", "", abs).Start()
	case "darwin":
		return exec.Command("open", abs).Start()
	case "linux", "freebsd", "openbsd", "netbsd":
		return exec.Command("xdg-open", abs).Start()
	default:
		return fmt.Errorf("open with default app: unsupported OS %q", runtime.GOOS)
	}
}

func (s *Similarity) absPathUnderRootFromRel(relPath string) (string, error) {
	relPath = strings.TrimSpace(relPath)
	if relPath == "" {
		return "", fmt.Errorf("empty path")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.rootAbs == "" {
		return "", fmt.Errorf("no scan root: open in file manager requires a folder scan with a root on disk (not JSON-only without root)")
	}
	return s.absPathUnderRootUnlocked(relPath)
}

// OpenScannedPath opens a scan-root-relative path (matches core File/Folder.Path) using the OS default handler.
func (s *Similarity) OpenScannedPath(relPath string) error {
	abs, err := s.absPathUnderRootFromRel(relPath)
	if err != nil {
		return err
	}
	return openAbsWithDefaultApp(abs)
}

// RevealScannedPath opens the OS file manager for the path; for files, selects the file in its parent folder on macOS and Windows.
func (s *Similarity) RevealScannedPath(relPath string) error {
	abs, err := s.absPathUnderRootFromRel(relPath)
	if err != nil {
		return err
	}
	return revealAbsInOS(abs)
}

func (s *Similarity) absPathUnderRootUnlocked(rel string) (string, error) {
	rel = pathutil.NormalizeRelStorageKey(strings.TrimSpace(rel))
	if rel == "" {
		rel = "."
	}
	if rel != "." && (strings.HasPrefix(rel, string(filepath.Separator)) || strings.Contains(rel, "..")) {
		return "", fmt.Errorf("invalid path")
	}
	abs := rel
	if rel != "." {
		abs = filepath.Join(s.rootAbs, rel)
	} else {
		abs = s.rootAbs
	}
	abs, err := filepath.Abs(abs)
	if err != nil {
		return "", err
	}
	if !pathutil.IsRootOrUnder(s.rootAbs, abs) {
		return "", fmt.Errorf("path outside scan root")
	}
	return abs, nil
}

func revealAbsInOS(abs string) error {
	abs = filepath.Clean(abs)
	st, err := os.Stat(abs)
	if err != nil {
		return err
	}
	switch runtime.GOOS {
	case "windows":
		if st.IsDir() {
			return exec.Command("explorer", abs).Start()
		}
		// /select, opens parent window with file selected
		return exec.Command("explorer", "/select,"+abs).Start()
	case "darwin":
		if st.IsDir() {
			return exec.Command("open", abs).Start()
		}
		return exec.Command("open", "-R", abs).Start()
	case "linux", "freebsd", "openbsd", "netbsd":
		// xdg-open opens dir or file in default app; good enough for “show in file manager” on many distros
		return exec.Command("xdg-open", abs).Start()
	default:
		return fmt.Errorf("reveal in file manager: unsupported OS %q", runtime.GOOS)
	}
}
