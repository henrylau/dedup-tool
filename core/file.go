// Package core provides file similarity detection and folder comparison functionality.
// It includes storage management, file hashing, and duplicate detection algorithms.
package core

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kalafut/imohash"
)

var ErrNotEmptyFolder = errors.New("folder is not empty")

// FileHash computes a hash for the given file using the provided hash algorithm.
func FileHash(file fs.File, hash imohash.ImoHash) (string, error) {
	readerAt, ok := file.(io.ReaderAt)
	if !ok {
		fmt.Println("File does not implement io.ReaderAt")
		return "", fmt.Errorf("file does not implement io.ReaderAt")
	}

	fi, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to get file size: %w", err)
	}

	hashValue, err := hash.SumSectionReader(io.NewSectionReader(readerAt, 0, fi.Size()))
	if err != nil {
		return "", fmt.Errorf("failed to hash file: %w", err)
	}

	return base64.RawStdEncoding.EncodeToString(hashValue[:]), nil

}

// ScanFolder recursively scans a directory and adds all files to storage.
func ScanFolder(ctx context.Context, path string, storage Storage) error {
	dirFS := os.DirFS(path)

	return fs.WalkDir(dirFS, ".", func(path string, d fs.DirEntry, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			return err
		}
		if d.IsDir() || d.Name()[0] == '.' {
			return nil
		}

		f, err := dirFS.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", path, err)
		}
		defer f.Close()

		stats, err := f.Stat()
		if err != nil {
			return fmt.Errorf("failed to stat file %s: %w", path, err)
		}

		hash, err := FileHash(f, imohash.New())
		if err != nil {
			return fmt.Errorf("failed to hash file %s: %w", path, err)
		}

		storage.AddFile(&File{
			Path:    path,
			Hash:    hash,
			Size:    stats.Size(),
			ModTime: stats.ModTime(),
			Name:    stats.Name(),
		})

		return nil
	})
}

// FormatFileSize formats a file size in bytes into a human-readable string.
func FormatFileSize(size int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	s := float64(size)
	i := 0
	for s >= 1024 && i < len(units)-1 {
		s /= 1024
		i++
	}
	return fmt.Sprintf("%.2f%s", s, units[i])
}

// FileAction represents the type of action to perform on a file.
type FileAction int

const (
	Move FileAction = iota
	Delete
	MoveFolder
	DeleteFolder
	DeleteEmptyFolder
)

// FileActionTask represents a task to perform on a file.
type FileActionTask struct {
	Action       FileAction
	File         *File
	Folder       *Folder
	TargetFolder *Folder
	TargetName   string
	NotDuplicate bool
}

func (f *FileActionTask) String() string {
	switch f.Action {
	case Move:
		targetName := f.TargetName
		if targetName == "" {
			targetName = f.File.Name
		}
		return fmt.Sprintf("move %s to %s", f.File.Path, filepath.Join(f.TargetFolder.Path, targetName))
	case Delete:
		return fmt.Sprintf("delete %s", f.File.Path)
	case MoveFolder:
		return fmt.Sprintf("move folder %s to %s", f.Folder.Path, f.TargetFolder.Path)
	case DeleteFolder:
		// fmt.Printf("Delete folder: %#v\n", f)
		return fmt.Sprintf("Delete folder: %s", f.Folder.Path)
	case DeleteEmptyFolder:
		return fmt.Sprintf("Delete empty folder: %s", f.Folder.Path)
	}
	return ""
}

// ExecuteFileActionTask executes a file action task.
func ExecuteFileActionTask(storage Storage, root *os.Root, task *FileActionTask) error {
	switch task.Action {
	case Move:
		exists := false
		if f, err := root.Open(filepath.Join(task.TargetFolder.Path, task.TargetName)); err == nil {
			f.Close()
			exists = true
		}
		targetName := task.TargetName
		if targetName == "" {
			targetName = task.File.Name
		}

		err := root.Rename(task.File.Path, filepath.Join(task.TargetFolder.Path, targetName))
		if err != nil {
			return err
		}

		err = storage.RemoveFile(task.File)
		if err != nil {
			return err
		}

		if !exists {
			storage.AddFile(&File{
				Path:    filepath.Join(task.TargetFolder.Path, targetName),
				Hash:    task.File.Hash,
				Size:    task.File.Size,
				ModTime: task.File.ModTime,
				Name:    targetName,
			})
		}
		return nil
	case Delete:
		err := root.Remove(task.File.Path)
		if err != nil {
			return err
		}
		return storage.RemoveFile(task.File)
	case MoveFolder:
		if task.Folder == nil {
			return fmt.Errorf("folder is nil")
		}
		targetPath := filepath.Join(task.TargetFolder.Path, task.Folder.Name)

		// if target folder already exists, return error
		if _, err := root.Stat(targetPath); err == nil {
			return fmt.Errorf("target folder %s already exists", targetPath)
		}

		err := root.Rename(task.Folder.Path, targetPath)
		if err != nil {
			return err
		}

		// TODO: remove folder from storage

		return nil
	case DeleteFolder:
		if task.Folder == nil {
			return fmt.Errorf("folder is nil")
		}
		err := root.Remove(task.Folder.Path)
		if err != nil {
			return err
		}

		// TODO: remove folder from storage

		return nil
	case DeleteEmptyFolder:
		if task.Folder == nil {
			return fmt.Errorf("folder is nil")
		}
		return RemoveEmptyFolder(root, task.Folder.Path)
	default:
		return nil
	}
}

func RemoveEmptyFolder(root *os.Root, path string) error {
	dir, err := root.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open dir %s: %w", path, err)
	}
	defer dir.Close()

	entries, err := dir.Readdir(0)
	if err != nil {
		return fmt.Errorf("failed to read dir %s: %w", path, err)
	}

	// Check if folder is empty (only contains hidden files like .DS_Store)
	// A folder is considered empty if it only contains hidden files (starting with ".")
	// Subdirectories (hidden or not) mean the folder is not empty
	isEmpty := true
	for _, entry := range entries {
		name := entry.Name()
		if name == "." || name == ".." {
			continue
		}

		// If entry is a subdirectory, folder is not empty
		if entry.IsDir() {
			isEmpty = false
			break
		}

		// If entry is not a hidden file, treat folder as non-empty
		if !strings.HasPrefix(name, ".") {
			isEmpty = false
			break
		}
	}

	if !isEmpty {
		return ErrNotEmptyFolder
	}

	if err := root.RemoveAll(path); err != nil {
		return fmt.Errorf("failed to remove empty folder %s: %w", path, err)
	}

	return nil
}

// Logger interface for logging messages.
type Logger interface {
	Info(message string)
	Error(message string)
}

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
