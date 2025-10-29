// Package core provides file similarity detection and folder comparison functionality.
// It includes storage management, file hashing, and duplicate detection algorithms.
package core

import (
	"fmt"
	"path/filepath"
	"slices"
	"sync"
)

// Storage interface defines methods for storing and retrieving file and folder data.
type Storage interface {
	AddFile(file *File) error
	GetFolder(path string) (*Folder, error)
	GetMatchedFiles() ([]*MatchedFileGroup, error)
	RemoveFile(file *File) error
}

// MemoryStorage implements Storage using in-memory data structures.
type MemoryStorage struct {
	folders      sync.Map
	matchedFiles sync.Map
	hashMap      sync.Map
}

var _ Storage = &MemoryStorage{}

// RemoveFile removes a file from storage.
func (s *MemoryStorage) RemoveFile(file *File) error {

	parentFolder, err := s.GetFolder(filepath.Dir(file.Path))
	if err != nil {
		return err
	}
	parentFolder.RemoveFile(file)

	if matchedPair, ok := s.matchedFiles.Load(file.Hash); ok {
		pair := matchedPair.(*MatchedFileGroup)
		pair.Files = slices.DeleteFunc(pair.Files, func(f *File) bool {
			return f == file
		})

		// Should not happen, but just in case
		if len(pair.Files) == 0 {
			s.matchedFiles.Delete(file.Hash)
			s.hashMap.Delete(file.Hash)
			return fmt.Errorf("internal error: matched file group became empty for hash %s", file.Hash)
		} else if len(pair.Files) == 1 {
			s.matchedFiles.Delete(file.Hash)
		}
		s.hashMap.Store(file.Hash, pair.Files[0])
	} else {
		s.hashMap.Delete(file.Hash)
	}

	return nil
}

// AddFile adds a file to storage.
func (s *MemoryStorage) AddFile(file *File) error {
	parentFolder, err := s.GetFolder(filepath.Dir(file.Path))
	if err != nil {
		return err
	}

	parentFolder.AddFile(file)

	// Record the file hash to fileHashMap
	if matchedFile, ok := s.hashMap.Load(file.Hash); !ok {
		s.hashMap.Store(file.Hash, file)
	} else {
		// append file to matched group
		if matchedPair, ok := s.matchedFiles.Load(file.Hash); ok {
			matchedPair.(*MatchedFileGroup).Files = append(matchedPair.(*MatchedFileGroup).Files, file)
		} else {
			// new MatchedFile
			newMatchedFile := &MatchedFileGroup{
				Files: []*File{file, matchedFile.(*File)},
				Hash:  file.Hash,
			}
			s.matchedFiles.Store(file.Hash, newMatchedFile)
		}
	}

	return nil
}

// GetFolder retrieves a folder by path, creating it if it doesn't exist.
func (s *MemoryStorage) GetFolder(path string) (*Folder, error) {
	if f, ok := s.folders.Load(path); ok {
		return f.(*Folder), nil
	} else if path == "." || path == "/" {
		rootFolder := &Folder{
			Name: path,
			Path: path,
		}

		s.folders.Store(path, rootFolder)
		return rootFolder, nil
	}

	parentPath := filepath.Dir(path)
	parentFolder, err := s.GetFolder(parentPath)
	if err != nil {
		return nil, err
	}

	newFolder := &Folder{
		Name:   filepath.Base(path),
		Path:   path,
		Parent: parentFolder,
	}
	parentFolder.Folders.Store(filepath.Base(path), newFolder)

	// Store new folder in memory storage
	s.folders.Store(path, newFolder)
	return newFolder, nil
}

// GetMatchedFiles returns all groups of files with matching hashes.
func (s *MemoryStorage) GetMatchedFiles() ([]*MatchedFileGroup, error) {
	matchedFiles := []*MatchedFileGroup{}

	s.matchedFiles.Range(func(key, value interface{}) bool {
		matchedFiles = append(matchedFiles, value.(*MatchedFileGroup))
		return true
	})

	return matchedFiles, nil
}

// NewMemoryStorage creates a new memory storage instance.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{}
}
