package core

import (
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// Test helper functions
func createTestFile(name, path, hash string, size int64) *File {
	return &File{
		Name:    name,
		Path:    path,
		Hash:    hash,
		Size:    size,
		ModTime: time.Now(),
	}
}

func createTestFolder(t *testing.T, path string) *Folder {
	folder := &Folder{Name: filepath.Base(path), Path: path}
	return folder
}

func TestFolder_AddFile(t *testing.T) {
	folder := &Folder{Name: "test", Path: "test"}
	file := createTestFile("file1.txt", "test/file1.txt", "hash1", 1024)

	err := folder.AddFile(file)
	if err != nil {
		t.Fatalf("AddFile failed: %v", err)
	}

	if file.Parent != folder {
		t.Errorf("Expected file.Parent to be %v, got %v", folder, file.Parent)
	}

	if folder.GetFileCount() != 1 {
		t.Errorf("Expected file count 1, got %d", folder.GetFileCount())
	}

	files := folder.GetFiles()
	if len(files) != 1 || files[0] != file {
		t.Errorf("Expected files to contain the added file")
	}
}

func TestFolder_RemoveFile(t *testing.T) {
	folder := &Folder{Name: "test", Path: "test"}
	file := createTestFile("file1.txt", "test/file1.txt", "hash1", 1024)

	// Add file first
	folder.AddFile(file)
	if folder.GetFileCount() != 1 {
		t.Errorf("Expected file count 1 after add, got %d", folder.GetFileCount())
	}

	// Remove file
	err := folder.RemoveFile(file)
	if err != nil {
		t.Fatalf("RemoveFile failed: %v", err)
	}

	if file.Parent != nil {
		t.Errorf("Expected file.Parent to be nil after removal, got %v", file.Parent)
	}

	if folder.GetFileCount() != 0 {
		t.Errorf("Expected file count 0 after removal, got %d", folder.GetFileCount())
	}

	files := folder.GetFiles()
	if len(files) != 0 {
		t.Errorf("Expected no files after removal, got %d", len(files))
	}
}

func TestFolder_GetFiles(t *testing.T) {
	folder := &Folder{Name: "test", Path: "test"}
	file1 := createTestFile("file1.txt", "test/file1.txt", "hash1", 1024)
	file2 := createTestFile("file2.txt", "test/file2.txt", "hash2", 2048)

	folder.AddFile(file1)
	folder.AddFile(file2)

	files := folder.GetFiles()
	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}

	// Check that both files are present
	found1, found2 := false, false
	for _, f := range files {
		if f == file1 {
			found1 = true
		}
		if f == file2 {
			found2 = true
		}
	}

	if !found1 || !found2 {
		t.Errorf("Expected both files to be found in GetFiles()")
	}
}

func TestFolder_GetFileCount(t *testing.T) {
	folder := &Folder{Name: "test", Path: "test"}
	file1 := createTestFile("file1.txt", "test/file1.txt", "hash1", 1024)
	file2 := createTestFile("file2.txt", "test/file2.txt", "hash2", 2048)

	// Test empty folder
	if folder.GetFileCount() != 0 {
		t.Errorf("Expected file count 0 for empty folder, got %d", folder.GetFileCount())
	}

	// Add files
	folder.AddFile(file1)
	if folder.GetFileCount() != 1 {
		t.Errorf("Expected file count 1 after adding first file, got %d", folder.GetFileCount())
	}

	folder.AddFile(file2)
	if folder.GetFileCount() != 2 {
		t.Errorf("Expected file count 2 after adding second file, got %d", folder.GetFileCount())
	}

	// Remove a file
	folder.RemoveFile(file1)
	if folder.GetFileCount() != 1 {
		t.Errorf("Expected file count 1 after removing first file, got %d", folder.GetFileCount())
	}
}

func TestFolder_GetDirectFileCount(t *testing.T) {
	root := &Folder{Name: "root", Path: "root"}
	child := &Folder{Name: "child", Path: "root/child", Parent: root}
	root.Folders.Store(child.Name, child)

	root.AddFile(createTestFile("a.txt", "root/a.txt", "h1", 1))
	child.AddFile(createTestFile("b.txt", "root/child/b.txt", "h2", 1))

	if root.GetDirectFileCount() != 1 {
		t.Errorf("expected root GetDirectFileCount 1, got %d", root.GetDirectFileCount())
	}
	if root.GetFileCount() != 2 {
		t.Errorf("expected root GetFileCount 2, got %d", root.GetFileCount())
	}
	if child.GetDirectFileCount() != 1 {
		t.Errorf("expected child GetDirectFileCount 1, got %d", child.GetDirectFileCount())
	}
}

func TestFolder_GetDirectFilesSize_GetTotalFilesSizeRecursive(t *testing.T) {
	root := &Folder{Name: "root", Path: "root"}
	child := &Folder{Name: "child", Path: "root/child", Parent: root}
	root.Folders.Store(child.Name, child)

	root.AddFile(createTestFile("a.txt", "root/a.txt", "h1", 100))
	root.AddFile(createTestFile("c.txt", "root/c.txt", "h3", 400))
	child.AddFile(createTestFile("b.txt", "root/child/b.txt", "h2", 50))

	if root.GetDirectFilesSize() != 500 {
		t.Errorf("expected root GetDirectFilesSize 500, got %d", root.GetDirectFilesSize())
	}
	if child.GetDirectFilesSize() != 50 {
		t.Errorf("expected child GetDirectFilesSize 50, got %d", child.GetDirectFilesSize())
	}
	if root.GetTotalFilesSizeRecursive() != 550 {
		t.Errorf("expected root GetTotalFilesSizeRecursive 550, got %d", root.GetTotalFilesSizeRecursive())
	}
}

func TestFolder_GetFileCount_Cache(t *testing.T) {
	folder := &Folder{Name: "test", Path: "test"}
	file1 := createTestFile("file1.txt", "test/file1.txt", "hash1", 1024)

	// Add file to populate cache
	folder.AddFile(file1)
	count1 := folder.GetFileCount()
	count2 := folder.GetFileCount()

	// Both calls should return the same value (cached)
	if count1 != count2 {
		t.Errorf("Expected cached file count to be consistent, got %d and %d", count1, count2)
	}

	if count1 != 1 {
		t.Errorf("Expected cached file count 1, got %d", count1)
	}
}

func TestFolder_GetFileCount_CacheInvalidation(t *testing.T) {
	folder := &Folder{Name: "test", Path: "test"}
	file1 := createTestFile("file1.txt", "test/file1.txt", "hash1", 1024)
	file2 := createTestFile("file2.txt", "test/file2.txt", "hash2", 2048)

	// Add first file and get count (populates cache)
	folder.AddFile(file1)
	count1 := folder.GetFileCount()

	// Add second file (should invalidate cache)
	folder.AddFile(file2)
	count2 := folder.GetFileCount()

	if count1 != 1 {
		t.Errorf("Expected first count to be 1, got %d", count1)
	}
	if count2 != 2 {
		t.Errorf("Expected second count to be 2, got %d", count2)
	}

	// Remove file (should invalidate cache again)
	folder.RemoveFile(file1)
	count3 := folder.GetFileCount()

	if count3 != 1 {
		t.Errorf("Expected count after removal to be 1, got %d", count3)
	}
}

func TestFolder_GetFolders(t *testing.T) {
	parent := &Folder{Name: "parent", Path: "parent"}
	child1 := &Folder{Name: "child1", Path: "parent/child1", Parent: parent}
	child2 := &Folder{Name: "child2", Path: "parent/child2", Parent: parent}

	parent.Folders.Store("child1", child1)
	parent.Folders.Store("child2", child2)

	folders := parent.GetFolders()
	if len(folders) != 2 {
		t.Errorf("Expected 2 folders, got %d", len(folders))
	}

	// Check that both folders are present
	found1, found2 := false, false
	for _, f := range folders {
		if f == child1 {
			found1 = true
		}
		if f == child2 {
			found2 = true
		}
	}

	if !found1 || !found2 {
		t.Errorf("Expected both folders to be found in GetFolders()")
	}
}

func TestFolder_Concurrent(t *testing.T) {
	folder := &Folder{Name: "test", Path: "test"}
	var wg sync.WaitGroup
	numGoroutines := 10
	filesPerGoroutine := 5

	// Test concurrent AddFile
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < filesPerGoroutine; j++ {
				file := createTestFile(
					"file_"+string(rune(goroutineID))+"_"+string(rune(j))+".txt",
					filepath.Join("test", "file_"+string(rune(goroutineID))+"_"+string(rune(j))+".txt"),
					"hash_"+string(rune(goroutineID))+"_"+string(rune(j)),
					1024,
				)
				folder.AddFile(file)
			}
		}(i)
	}

	wg.Wait()

	expectedCount := numGoroutines * filesPerGoroutine
	if folder.GetFileCount() != expectedCount {
		t.Errorf("Expected file count %d, got %d", expectedCount, folder.GetFileCount())
	}

	// Test concurrent GetFileCount
	var countResults []int
	var countMutex sync.Mutex
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			count := folder.GetFileCount()
			countMutex.Lock()
			countResults = append(countResults, count)
			countMutex.Unlock()
		}()
	}

	wg.Wait()

	// All counts should be the same
	for _, count := range countResults {
		if count != expectedCount {
			t.Errorf("Expected all concurrent counts to be %d, got %d", expectedCount, count)
		}
	}
}

func TestFolder_GetFileCount_DeepNesting(t *testing.T) {
	// Create a deep folder structure
	root := &Folder{Name: "root", Path: "root"}
	current := root

	// Create 5 levels of nesting
	for i := 0; i < 5; i++ {
		child := &Folder{
			Name:   "level" + string(rune(i)),
			Path:   filepath.Join(current.Path, "level"+string(rune(i))),
			Parent: current,
		}
		current.Folders.Store(child.Name, child)
		current = child
	}

	// Add files at each level
	current = root
	for i := 0; i < 5; i++ {
		file := createTestFile(
			"file"+string(rune(i))+".txt",
			filepath.Join(current.Path, "file"+string(rune(i))+".txt"),
			"hash"+string(rune(i)),
			1024,
		)
		current.AddFile(file)

		if i < 4 {
			// Move to next level
			if child, ok := current.Folders.Load("level" + string(rune(i))); ok {
				current = child.(*Folder)
			}
		}
	}

	// Root should have count of 5 (all files)
	if root.GetFileCount() != 5 {
		t.Errorf("Expected root file count 5, got %d", root.GetFileCount())
	}

	// Each level should have count of files at that level and below
	current = root
	for i := 0; i < 5; i++ {
		expectedCount := 5 - i
		if current.GetFileCount() != expectedCount {
			t.Errorf("Expected level %d file count %d, got %d", i, expectedCount, current.GetFileCount())
		}

		if i < 4 {
			if child, ok := current.Folders.Load("level" + string(rune(i))); ok {
				current = child.(*Folder)
			}
		}
	}
}

func TestFolder_invalidateCache(t *testing.T) {
	parent := &Folder{Name: "parent", Path: "parent"}
	child := &Folder{Name: "child", Path: "parent/child", Parent: parent}

	// Add files to both folders
	parentFile := createTestFile("parent.txt", "parent/parent.txt", "hash1", 1024)
	childFile := createTestFile("child.txt", "parent/child/child.txt", "hash2", 2048)

	parent.AddFile(parentFile)
	child.AddFile(childFile)

	// Get counts to populate cache
	parentCount := parent.GetFileCount()
	childCount := child.GetFileCount()

	if parentCount != 1 { // only parent file
		t.Errorf("Expected parent count 1, got %d", parentCount)
	}
	if childCount != 1 {
		t.Errorf("Expected child count 1, got %d", childCount)
	}

	// Add file to child (should invalidate parent cache)
	childFile2 := createTestFile("child2.txt", "parent/child/child2.txt", "hash3", 1024)
	child.AddFile(childFile2)

	// Parent count should still be 1 (only counts direct files)
	newParentCount := parent.GetFileCount()
	if newParentCount != 1 {
		t.Errorf("Expected parent count 1 after child addition, got %d", newParentCount)
	}
}
