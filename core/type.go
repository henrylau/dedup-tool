// Package core provides file similarity detection and folder comparison functionality.
// It includes storage management, file hashing, and duplicate detection algorithms.
package core

import (
	"sync"
	"sync/atomic"
	"time"
)

// File represents a file with metadata.
type File struct {
	Name    string
	Path    string
	Hash    string
	Size    int64
	Parent  *Folder
	ModTime time.Time
}

// Folder represents a folder with files and subfolders.
type Folder struct {
	Name           string
	Path           string
	Parent         *Folder
	Folders        sync.Map
	files          sync.Map
	fileCount      int32
	fileCountCache int32
}

// MatchedFileGroup represents a group of files with the same hash.
type MatchedFileGroup struct {
	Files []*File
	Hash  string
}

// GetFiles returns all files in this folder.
func (f *Folder) GetFiles() []*File {
	files := []*File{}
	f.files.Range(func(key, value interface{}) bool {
		files = append(files, value.(*File))
		return true
	})
	return files
}

// AddFile adds a file to this folder.
func (f *Folder) AddFile(file *File) error {
	f.files.Store(file.Name, file)
	file.Parent = f
	atomic.AddInt32(&f.fileCount, 1)
	f.invalidateCache()
	return nil
}

// RemoveFile removes a file from this folder.
func (f *Folder) RemoveFile(file *File) error {
	f.files.Delete(file.Name)
	file.Parent = nil
	atomic.AddInt32(&f.fileCount, -1)
	f.invalidateCache()
	return nil
}

// GetFileCount returns the total number of files in this folder and subfolders.
func (f *Folder) GetFileCount() int {
	cached := atomic.LoadInt32(&f.fileCountCache)
	if cached != 0 {
		return int(cached)
	}

	// calculate file count recursively
	c := int(atomic.LoadInt32(&f.fileCount))
	f.Folders.Range(func(key, value interface{}) bool {
		c += value.(*Folder).GetFileCount()
		return true
	})
	atomic.StoreInt32(&f.fileCountCache, int32(c))
	return c
}

// GetFolders returns all subfolders of this folder.
func (f *Folder) GetFolders() []*Folder {
	folders := []*Folder{}

	f.Folders.Range(func(key, value interface{}) bool {
		folders = append(folders, value.(*Folder))
		return true
	})
	return folders
}

// invalidateCache clears the file count cache for this folder and its parents.
func (f *Folder) invalidateCache() {
	atomic.StoreInt32(&f.fileCountCache, 0)
	if f.Parent != nil {
		f.Parent.invalidateCache()
	}
}
