package core

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestMemoryStorage_AddFile(t *testing.T) {
	storage := NewMemoryStorageWithDupTrackingSkipRules([]string{})
	file := &File{
		Name:    "test.txt",
		Path:    "folder/test.txt",
		Hash:    "abc123",
		Size:    1024,
		ModTime: time.Now(),
	}

	err := storage.AddFile(file)
	if err != nil {
		t.Fatalf("AddFile failed: %v", err)
	}

	folder, err := storage.GetFolder("folder")
	if err != nil {
		t.Fatalf("GetFolder failed: %v", err)
	}

	files := folder.GetFiles()
	if len(files) != 1 || files[0] != file {
		t.Errorf("Expected folder to contain the added file")
	}

	if file.Parent != folder {
		t.Errorf("Expected file.Parent to be the folder")
	}
}

func TestMemoryStorage_AddFile_Multiple(t *testing.T) {
	storage := NewMemoryStorageWithDupTrackingSkipRules([]string{})
	files := []*File{
		{Name: "file1.txt", Path: "folder/file1.txt", Hash: "hash1", Size: 1024, ModTime: time.Now()},
		{Name: "file2.txt", Path: "folder/file2.txt", Hash: "hash2", Size: 2048, ModTime: time.Now()},
		{Name: "file3.txt", Path: "folder/file3.txt", Hash: "hash3", Size: 4096, ModTime: time.Now()},
	}

	for _, file := range files {
		err := storage.AddFile(file)
		if err != nil {
			t.Fatalf("AddFile failed for %s: %v", file.Name, err)
		}
	}

	folder, err := storage.GetFolder("folder")
	if err != nil {
		t.Fatalf("GetFolder failed: %v", err)
	}

	if folder.GetFileCount() != 3 {
		t.Errorf("Expected folder to have 3 files, got %d", folder.GetFileCount())
	}

	matchedFiles, err := storage.GetMatchedFiles()
	if err != nil {
		t.Fatalf("GetMatchedFiles failed: %v", err)
	}

	// Should have no matched files since all hashes are unique
	if len(matchedFiles) != 0 {
		t.Errorf("Expected no matched files, got %d", len(matchedFiles))
	}
}

func TestMemoryStorage_AddFile_Duplicates(t *testing.T) {
	storage := NewMemoryStorageWithDupTrackingSkipRules([]string{})

	// Add files with same hash
	file1 := &File{Name: "file1.txt", Path: "folder1/file1.txt", Hash: "samehash", Size: 1024, ModTime: time.Now()}
	file2 := &File{Name: "file2.txt", Path: "folder2/file2.txt", Hash: "samehash", Size: 1024, ModTime: time.Now()}

	err := storage.AddFile(file1)
	if err != nil {
		t.Fatalf("AddFile failed for file1: %v", err)
	}

	err = storage.AddFile(file2)
	if err != nil {
		t.Fatalf("AddFile failed for file2: %v", err)
	}

	matchedFiles, err := storage.GetMatchedFiles()
	if err != nil {
		t.Fatalf("GetMatchedFiles failed: %v", err)
	}

	if len(matchedFiles) != 1 {
		t.Errorf("Expected 1 matched file group, got %d", len(matchedFiles))
	}

	if len(matchedFiles[0].Files) != 2 {
		t.Errorf("Expected matched group to have 2 files, got %d", len(matchedFiles[0].Files))
	}

	if matchedFiles[0].Hash != "samehash" {
		t.Errorf("Expected hash 'samehash', got '%s'", matchedFiles[0].Hash)
	}
}

func TestMemoryStorage_RemoveFile(t *testing.T) {
	storage := NewMemoryStorageWithDupTrackingSkipRules([]string{})
	file := &File{Name: "test.txt", Path: "folder/test.txt", Hash: "abc123", Size: 1024, ModTime: time.Now()}

	// Add file first
	err := storage.AddFile(file)
	if err != nil {
		t.Fatalf("AddFile failed: %v", err)
	}

	// Verify file exists
	folder, err := storage.GetFolder("folder")
	if err != nil {
		t.Fatalf("GetFolder failed: %v", err)
	}
	if folder.GetFileCount() != 1 {
		t.Errorf("Expected folder to have 1 file before removal, got %d", folder.GetFileCount())
	}

	// Remove file
	err = storage.RemoveFile(file)
	if err != nil {
		t.Fatalf("RemoveFile failed: %v", err)
	}

	// Verify file is removed
	if folder.GetFileCount() != 0 {
		t.Errorf("Expected folder to have 0 files after removal, got %d", folder.GetFileCount())
	}

	if file.Parent != nil {
		t.Errorf("Expected file.Parent to be nil after removal")
	}
}

func TestMemoryStorage_RemoveFile_FromMatchedGroup(t *testing.T) {
	storage := NewMemoryStorageWithDupTrackingSkipRules([]string{})

	// Add files with same hash
	file1 := &File{Name: "file1.txt", Path: "folder1/file1.txt", Hash: "samehash", Size: 1024, ModTime: time.Now()}
	file2 := &File{Name: "file2.txt", Path: "folder2/file2.txt", Hash: "samehash", Size: 1024, ModTime: time.Now()}

	storage.AddFile(file1)
	storage.AddFile(file2)

	// Verify matched group exists
	matchedFiles, err := storage.GetMatchedFiles()
	if err != nil {
		t.Fatalf("GetMatchedFiles failed: %v", err)
	}
	if len(matchedFiles) != 1 || len(matchedFiles[0].Files) != 2 {
		t.Errorf("Expected 1 matched group with 2 files")
	}

	// Remove one file from the matched group
	err = storage.RemoveFile(file1)
	if err != nil {
		t.Fatalf("RemoveFile failed: %v", err)
	}

	// Verify matched group no longer exists (only 1 file left)
	matchedFiles, err = storage.GetMatchedFiles()
	if err != nil {
		t.Fatalf("GetMatchedFiles failed: %v", err)
	}
	if len(matchedFiles) != 0 {
		t.Errorf("Expected no matched groups after removing one file, got %d", len(matchedFiles))
	}
}

func TestMemoryStorage_GetFolder(t *testing.T) {
	storage := NewMemoryStorageWithDupTrackingSkipRules([]string{})

	// Test getting root folder
	root, err := storage.GetFolder(".")
	if err != nil {
		t.Fatalf("GetFolder failed for root: %v", err)
	}
	if root.Name != "." || root.Path != "." {
		t.Errorf("Expected root folder to have name '.' and path '.', got name '%s' and path '%s'", root.Name, root.Path)
	}

	// Test getting root folder with "/"
	rootSlash, err := storage.GetFolder("/")
	if err != nil {
		t.Fatalf("GetFolder failed for root slash: %v", err)
	}
	if rootSlash.Name != "/" || rootSlash.Path != "/" {
		t.Errorf("Expected root slash folder to have name '/' and path '/', got name '%s' and path '%s'", rootSlash.Name, rootSlash.Path)
	}

	// Test getting non-existent folder (should create it)
	folder, err := storage.GetFolder("test/folder")
	if err != nil {
		t.Fatalf("GetFolder failed for test/folder: %v", err)
	}
	if folder.Name != "folder" || folder.Path != "test/folder" {
		t.Errorf("Expected folder to have name 'folder' and path 'test/folder', got name '%s' and path '%s'", folder.Name, folder.Path)
	}

	// Test getting parent folder
	parent, err := storage.GetFolder("test")
	if err != nil {
		t.Fatalf("GetFolder failed for test: %v", err)
	}
	if parent.Name != "test" || parent.Path != "test" {
		t.Errorf("Expected parent to have name 'test' and path 'test', got name '%s' and path '%s'", parent.Name, parent.Path)
	}

	// Verify parent-child relationship
	if folder.Parent != parent {
		t.Errorf("Expected folder.Parent to be the parent folder")
	}
}

func TestMemoryStorage_GetFolder_Nested(t *testing.T) {
	storage := NewMemoryStorageWithDupTrackingSkipRules([]string{})

	// Create deeply nested folder structure
	deepPath := "level1/level2/level3/level4/level5"
	folder, err := storage.GetFolder(deepPath)
	if err != nil {
		t.Fatalf("GetFolder failed for deep path: %v", err)
	}

	if folder.Name != "level5" || folder.Path != deepPath {
		t.Errorf("Expected folder to have name 'level5' and path '%s', got name '%s' and path '%s'", deepPath, folder.Name, folder.Path)
	}

	// Verify all parent folders were created
	current := folder
	expectedLevels := []string{"level5", "level4", "level3", "level2", "level1"}
	for i, expectedLevel := range expectedLevels {
		if current.Name != expectedLevel {
			t.Errorf("Expected level %d to have name '%s', got '%s'", i, expectedLevel, current.Name)
		}
		if current.Parent == nil && i < len(expectedLevels)-1 {
			t.Errorf("Expected level %d to have a parent", i)
		}
		if current.Parent != nil {
			current = current.Parent
		}
	}
}

func TestMemoryStorage_GetMatchedFiles(t *testing.T) {
	storage := NewMemoryStorageWithDupTrackingSkipRules([]string{})

	// Add files with duplicates
	files := []*File{
		{Name: "file1.txt", Path: "folder1/file1.txt", Hash: "hash1", Size: 1024, ModTime: time.Now()},
		{Name: "file2.txt", Path: "folder2/file2.txt", Hash: "hash1", Size: 1024, ModTime: time.Now()}, // duplicate
		{Name: "file3.txt", Path: "folder1/file3.txt", Hash: "hash2", Size: 2048, ModTime: time.Now()},
		{Name: "file4.txt", Path: "folder3/file4.txt", Hash: "hash2", Size: 2048, ModTime: time.Now()}, // duplicate
		{Name: "file5.txt", Path: "folder1/file5.txt", Hash: "hash3", Size: 4096, ModTime: time.Now()}, // unique
	}

	for _, file := range files {
		err := storage.AddFile(file)
		if err != nil {
			t.Fatalf("AddFile failed for %s: %v", file.Name, err)
		}
	}

	matchedFiles, err := storage.GetMatchedFiles()
	if err != nil {
		t.Fatalf("GetMatchedFiles failed: %v", err)
	}

	// Should have 2 matched groups (hash1 and hash2)
	if len(matchedFiles) != 2 {
		t.Errorf("Expected 2 matched file groups, got %d", len(matchedFiles))
	}

	// Verify each group has 2 files
	for _, group := range matchedFiles {
		if len(group.Files) != 2 {
			t.Errorf("Expected matched group to have 2 files, got %d", len(group.Files))
		}
		if group.Hash != "hash1" && group.Hash != "hash2" {
			t.Errorf("Expected hash to be 'hash1' or 'hash2', got '%s'", group.Hash)
		}
	}
}

func TestMemoryStorage_GetMatchedFiles_Empty(t *testing.T) {
	storage := NewMemoryStorageWithDupTrackingSkipRules([]string{})

	matchedFiles, err := storage.GetMatchedFiles()
	if err != nil {
		t.Fatalf("GetMatchedFiles failed: %v", err)
	}

	if len(matchedFiles) != 0 {
		t.Errorf("Expected no matched files for empty storage, got %d", len(matchedFiles))
	}
}

func TestMemoryStorage_Concurrent(t *testing.T) {
	storage := NewMemoryStorageWithDupTrackingSkipRules([]string{})
	var wg sync.WaitGroup
	numGoroutines := 10
	filesPerGoroutine := 5

	// Test concurrent AddFile
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < filesPerGoroutine; j++ {
				file := &File{
					Name:    "file_" + fmt.Sprintf("%d", goroutineID) + "_" + fmt.Sprintf("%d", j) + ".txt",
					Path:    filepath.Join("folder", "file_"+fmt.Sprintf("%d", goroutineID)+"_"+fmt.Sprintf("%d", j)+".txt"),
					Hash:    "hash_" + fmt.Sprintf("%d", goroutineID) + "_" + fmt.Sprintf("%d", j),
					Size:    1024,
					ModTime: time.Now(),
				}
				storage.AddFile(file)
			}
		}(i)
	}

	wg.Wait()

	// Verify all files were added
	folder, err := storage.GetFolder("folder")
	if err != nil {
		t.Fatalf("GetFolder failed: %v", err)
	}

	expectedCount := numGoroutines * filesPerGoroutine
	if folder.GetFileCount() != expectedCount {
		t.Errorf("Expected folder to have %d files, got %d", expectedCount, folder.GetFileCount())
	}

	// Test concurrent GetMatchedFiles
	var matchedResults [][]*MatchedFileGroup
	var resultsMutex sync.Mutex
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			matchedFiles, err := storage.GetMatchedFiles()
			if err != nil {
				t.Errorf("GetMatchedFiles failed: %v", err)
				return
			}
			resultsMutex.Lock()
			matchedResults = append(matchedResults, matchedFiles)
			resultsMutex.Unlock()
		}()
	}

	wg.Wait()

	// All results should be the same
	for i, result := range matchedResults {
		if len(result) != 0 {
			t.Errorf("Expected result %d to have 0 matched groups (all hashes unique), got %d", i, len(result))
		}
	}
}

func TestMemoryStorage_RemoveFile_LastInGroup(t *testing.T) {
	storage := NewMemoryStorageWithDupTrackingSkipRules([]string{})

	// Add files with same hash
	file1 := &File{Name: "file1.txt", Path: "folder1/file1.txt", Hash: "samehash", Size: 1024, ModTime: time.Now()}
	file2 := &File{Name: "file2.txt", Path: "folder2/file2.txt", Hash: "samehash", Size: 1024, ModTime: time.Now()}

	storage.AddFile(file1)
	storage.AddFile(file2)

	// Verify matched group exists
	matchedFiles, err := storage.GetMatchedFiles()
	if err != nil {
		t.Fatalf("GetMatchedFiles failed: %v", err)
	}
	if len(matchedFiles) != 1 || len(matchedFiles[0].Files) != 2 {
		t.Errorf("Expected 1 matched group with 2 files")
	}

	// Remove first file
	err = storage.RemoveFile(file1)
	if err != nil {
		t.Fatalf("RemoveFile failed for file1: %v", err)
	}

	// Remove second file (last in group)
	err = storage.RemoveFile(file2)
	if err != nil {
		t.Fatalf("RemoveFile failed for file2: %v", err)
	}

	// Verify no matched groups exist
	matchedFiles, err = storage.GetMatchedFiles()
	if err != nil {
		t.Fatalf("GetMatchedFiles failed: %v", err)
	}
	if len(matchedFiles) != 0 {
		t.Errorf("Expected no matched groups after removing all files, got %d", len(matchedFiles))
	}
}

func TestMemoryStorage_DefaultSkipsTxtLnkDupTracking(t *testing.T) {
	storage := NewMemoryStorage()
	file1 := &File{Name: "a.txt", Path: "folder/a.txt", Hash: "same", Size: 10, ModTime: time.Now()}
	file2 := &File{Name: "b.txt", Path: "folder2/b.txt", Hash: "same", Size: 10, ModTime: time.Now()}
	if err := storage.AddFile(file1); err != nil {
		t.Fatal(err)
	}
	if err := storage.AddFile(file2); err != nil {
		t.Fatal(err)
	}
	matched, err := storage.GetMatchedFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(matched) != 0 {
		t.Fatalf("expected no matched groups for .txt with default rules, got %d", len(matched))
	}
}

func TestMemoryStorage_AllowsTxtMatchedWhenDupSkipDisabled(t *testing.T) {
	storage := NewMemoryStorageWithDupTrackingSkipRules([]string{})
	file1 := &File{Name: "a.txt", Path: "folder/a.txt", Hash: "same", Size: 10, ModTime: time.Now()}
	file2 := &File{Name: "b.txt", Path: "folder2/b.txt", Hash: "same", Size: 10, ModTime: time.Now()}
	if err := storage.AddFile(file1); err != nil {
		t.Fatal(err)
	}
	if err := storage.AddFile(file2); err != nil {
		t.Fatal(err)
	}
	matched, err := storage.GetMatchedFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(matched) != 1 || len(matched[0].Files) != 2 {
		t.Fatalf("expected one matched pair with dup skip disabled, got %+v", matched)
	}
}

func TestMemoryStorage_CustomDupSkipRulePdfOnly(t *testing.T) {
	storage := NewMemoryStorageWithDupTrackingSkipRules([]string{".pdf"})
	pdf1 := &File{Name: "a.pdf", Path: "f1/a.pdf", Hash: "h1", Size: 1, ModTime: time.Now()}
	pdf2 := &File{Name: "b.pdf", Path: "f2/b.pdf", Hash: "h1", Size: 1, ModTime: time.Now()}
	go1 := &File{Name: "x.go", Path: "f1/x.go", Hash: "h2", Size: 1, ModTime: time.Now()}
	go2 := &File{Name: "y.go", Path: "f2/y.go", Hash: "h2", Size: 1, ModTime: time.Now()}
	for _, f := range []*File{pdf1, pdf2, go1, go2} {
		if err := storage.AddFile(f); err != nil {
			t.Fatal(err)
		}
	}
	matched, err := storage.GetMatchedFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(matched) != 1 || matched[0].Hash != "h2" {
		t.Fatalf("expected only .go pair in matched files, got %+v", matched)
	}
}
