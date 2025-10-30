package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileAction represents the type of action to perform on a file.
type FileAction int

const (
	Move FileAction = iota
	Delete
	MoveFolder
	DeleteFolder
	DeleteEmptyFolder
)

var ErrNotEmptyFolder = errors.New("folder is not empty")

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
