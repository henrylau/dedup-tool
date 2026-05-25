package core

import (
	"testing"
	"time"
)

func TestFolderSimilarity_DuplicatedPercentage(t *testing.T) {
	fs := &FolderSimilarity{
		FileCount:          10,
		DuplicateFileCount: 3,
	}

	percentage := fs.DuplicatedPercentage()
	expected := 30.0
	if percentage != expected {
		t.Errorf("Expected percentage %f, got %f", expected, percentage)
	}
}

func TestFolderSimilarity_DuplicatedPercentage_Zero(t *testing.T) {
	fs := &FolderSimilarity{
		FileCount:          5,
		DuplicateFileCount: 0,
	}

	percentage := fs.DuplicatedPercentage()
	expected := 0.0
	if percentage != expected {
		t.Errorf("Expected percentage %f, got %f", expected, percentage)
	}
}

func TestFolderSimilarity_DuplicatedPercentage_AllDuplicates(t *testing.T) {
	fs := &FolderSimilarity{
		FileCount:          4,
		DuplicateFileCount: 4,
	}

	percentage := fs.DuplicatedPercentage()
	expected := 100.0
	if percentage != expected {
		t.Errorf("Expected percentage %f, got %f", expected, percentage)
	}
}

func TestFolderPairKey(t *testing.T) {
	tests := []struct {
		path1    string
		path2    string
		expected string
	}{
		{"folder1", "folder2", "folder1:folder2"},
		{"folder2", "folder1", "folder1:folder2"}, // order should not matter
		{"a", "b", "a:b"},
		{"b", "a", "a:b"},
		{"same", "same", "same:same"},
	}

	for _, tt := range tests {
		result := folderPairKey(tt.path1, tt.path2)
		if result != tt.expected {
			t.Errorf("folderPairKey(%s, %s) = %s, expected %s", tt.path1, tt.path2, result, tt.expected)
		}
	}
}

func TestFolderPairKey_Order(t *testing.T) {
	path1 := "folder1"
	path2 := "folder2"

	key1 := folderPairKey(path1, path2)
	key2 := folderPairKey(path2, path1)

	if key1 != key2 {
		t.Errorf("Expected folderPairKey to be order-independent, got %s and %s", key1, key2)
	}
}

func TestSimilarityChecker_CalculateSimilarity(t *testing.T) {
	storage := NewMemoryStorageWithDupTrackingSkipRules([]string{})

	// Add files to folder1
	storage.AddFile(&File{Path: "folder1/file1.txt", Hash: "hash1", Name: "file1.txt", Size: 1024, ModTime: time.Now()})
	storage.AddFile(&File{Path: "folder1/file2.txt", Hash: "hash2", Name: "file2.txt", Size: 2048, ModTime: time.Now()})

	// Add files to folder2 (one duplicate)
	storage.AddFile(&File{Path: "folder2/file1.txt", Hash: "hash1", Name: "file1.txt", Size: 1024, ModTime: time.Now()})
	storage.AddFile(&File{Path: "folder2/file3.txt", Hash: "hash3", Name: "file3.txt", Size: 4096, ModTime: time.Now()})

	// Calculate similarity
	checker := &SimilarityChecker{}
	err := checker.CalculateSimilarity(storage)
	if err != nil {
		t.Fatalf("CalculateSimilarity failed: %v", err)
	}

	// Verify results
	if !checker.ContainsSimilarityGroup("folder1") {
		t.Errorf("Expected folder1 to have similarity group")
	}
	if !checker.ContainsSimilarityGroup("folder2") {
		t.Errorf("Expected folder2 to have similarity group")
	}

	// Get similarity groups
	groups1 := checker.GetSimilarityFolderGroup("folder1")
	if len(groups1) != 1 {
		t.Errorf("Expected folder1 to have 1 similarity group, got %d", len(groups1))
	}

	groups2 := checker.GetSimilarityFolderGroup("folder2")
	if len(groups2) != 1 {
		t.Errorf("Expected folder2 to have 1 similarity group, got %d", len(groups2))
	}

	// Verify duplicate file count
	if groups1[0][0].DuplicateFileCount != 1 {
		t.Errorf("Expected folder1 duplicate count 1, got %d", groups1[0][0].DuplicateFileCount)
	}
	if groups1[0][1].DuplicateFileCount != 1 {
		t.Errorf("Expected folder2 duplicate count 1, got %d", groups1[0][1].DuplicateFileCount)
	}
}

func TestSimilarityChecker_CalculateSimilarity_NoDuplicates(t *testing.T) {
	storage := NewMemoryStorageWithDupTrackingSkipRules([]string{})

	// Add files with unique hashes
	storage.AddFile(&File{Path: "folder1/file1.txt", Hash: "hash1", Name: "file1.txt", Size: 1024, ModTime: time.Now()})
	storage.AddFile(&File{Path: "folder1/file2.txt", Hash: "hash2", Name: "file2.txt", Size: 2048, ModTime: time.Now()})
	storage.AddFile(&File{Path: "folder2/file3.txt", Hash: "hash3", Name: "file3.txt", Size: 4096, ModTime: time.Now()})
	storage.AddFile(&File{Path: "folder2/file4.txt", Hash: "hash4", Name: "file4.txt", Size: 8192, ModTime: time.Now()})

	// Calculate similarity
	checker := &SimilarityChecker{}
	err := checker.CalculateSimilarity(storage)
	if err != nil {
		t.Fatalf("CalculateSimilarity failed: %v", err)
	}

	// Should have no similarity groups
	if checker.ContainsSimilarityGroup("folder1") {
		t.Errorf("Expected folder1 to have no similarity group")
	}
	if checker.ContainsSimilarityGroup("folder2") {
		t.Errorf("Expected folder2 to have no similarity group")
	}

	similarityFolders := checker.GetSimilarityFolder()
	if len(similarityFolders) != 0 {
		t.Errorf("Expected no similarity folders, got %d", len(similarityFolders))
	}
}

func TestSimilarityChecker_CalculateSimilarity_MultipleFolders(t *testing.T) {
	storage := NewMemoryStorageWithDupTrackingSkipRules([]string{})

	// Create multiple folders with overlapping duplicates
	// folder1: file1 (hash1), file2 (hash2)
	storage.AddFile(&File{Path: "folder1/file1.txt", Hash: "hash1", Name: "file1.txt", Size: 1024, ModTime: time.Now()})
	storage.AddFile(&File{Path: "folder1/file2.txt", Hash: "hash2", Name: "file2.txt", Size: 2048, ModTime: time.Now()})

	// folder2: file1 (hash1), file3 (hash3)
	storage.AddFile(&File{Path: "folder2/file1.txt", Hash: "hash1", Name: "file1.txt", Size: 1024, ModTime: time.Now()})
	storage.AddFile(&File{Path: "folder2/file3.txt", Hash: "hash3", Name: "file3.txt", Size: 4096, ModTime: time.Now()})

	// folder3: file2 (hash2), file4 (hash4)
	storage.AddFile(&File{Path: "folder3/file2.txt", Hash: "hash2", Name: "file2.txt", Size: 2048, ModTime: time.Now()})
	storage.AddFile(&File{Path: "folder3/file4.txt", Hash: "hash4", Name: "file4.txt", Size: 8192, ModTime: time.Now()})

	// Calculate similarity
	checker := &SimilarityChecker{}
	err := checker.CalculateSimilarity(storage)
	if err != nil {
		t.Fatalf("CalculateSimilarity failed: %v", err)
	}

	// All folders should have similarity groups
	if !checker.ContainsSimilarityGroup("folder1") {
		t.Errorf("Expected folder1 to have similarity group")
	}
	if !checker.ContainsSimilarityGroup("folder2") {
		t.Errorf("Expected folder2 to have similarity group")
	}
	if !checker.ContainsSimilarityGroup("folder3") {
		t.Errorf("Expected folder3 to have similarity group")
	}

	// Get similarity groups for each folder
	groups1 := checker.GetSimilarityFolderGroup("folder1")
	groups2 := checker.GetSimilarityFolderGroup("folder2")
	groups3 := checker.GetSimilarityFolderGroup("folder3")

	// Each folder should have 1 similarity group (one for each duplicate)
	if len(groups1) != 2 {
		t.Errorf("Expected folder1 to have 2 similarity groups, got %d", len(groups1))
	}
	if len(groups2) != 1 {
		t.Errorf("Expected folder2 to have 1 similarity group, got %d", len(groups2))
	}
	if len(groups3) != 1 {
		t.Errorf("Expected folder3 to have 1 similarity group, got %d", len(groups3))
	}
}

func TestSimilarityChecker_GetSimilarityFolder(t *testing.T) {
	storage := NewMemoryStorageWithDupTrackingSkipRules([]string{})

	// Add files with duplicates
	storage.AddFile(&File{Path: "folder1/file1.txt", Hash: "hash1", Name: "file1.txt", Size: 1024, ModTime: time.Now()})
	storage.AddFile(&File{Path: "folder2/file1.txt", Hash: "hash1", Name: "file1.txt", Size: 1024, ModTime: time.Now()})
	storage.AddFile(&File{Path: "folder3/file2.txt", Hash: "hash2", Name: "file2.txt", Size: 2048, ModTime: time.Now()})
	storage.AddFile(&File{Path: "folder4/file2.txt", Hash: "hash2", Name: "file2.txt", Size: 2048, ModTime: time.Now()})

	checker := &SimilarityChecker{}
	err := checker.CalculateSimilarity(storage)
	if err != nil {
		t.Fatalf("CalculateSimilarity failed: %v", err)
	}

	similarityFolders := checker.GetSimilarityFolder()
	if len(similarityFolders) != 4 {
		t.Errorf("Expected 4 similarity folders, got %d", len(similarityFolders))
	}

	// Check that all folders are present
	found := make(map[string]bool)
	for _, folder := range similarityFolders {
		found[folder] = true
	}

	expectedFolders := []string{"folder1", "folder2", "folder3", "folder4"}
	for _, expected := range expectedFolders {
		if !found[expected] {
			t.Errorf("Expected folder %s to be in similarity folders", expected)
		}
	}
}

func TestSimilarityChecker_GetSimilarityFolderGroup(t *testing.T) {
	storage := NewMemoryStorageWithDupTrackingSkipRules([]string{})

	// Add files with duplicates
	storage.AddFile(&File{Path: "folder1/file1.txt", Hash: "hash1", Name: "file1.txt", Size: 1024, ModTime: time.Now()})
	storage.AddFile(&File{Path: "folder1/file2.txt", Hash: "hash2", Name: "file2.txt", Size: 2048, ModTime: time.Now()})
	storage.AddFile(&File{Path: "folder2/file1.txt", Hash: "hash1", Name: "file1.txt", Size: 1024, ModTime: time.Now()})
	storage.AddFile(&File{Path: "folder3/file2.txt", Hash: "hash2", Name: "file2.txt", Size: 2048, ModTime: time.Now()})

	checker := &SimilarityChecker{}
	err := checker.CalculateSimilarity(storage)
	if err != nil {
		t.Fatalf("CalculateSimilarity failed: %v", err)
	}

	// Get groups for folder1
	groups1 := checker.GetSimilarityFolderGroup("folder1")
	if len(groups1) != 2 {
		t.Errorf("Expected folder1 to have 2 similarity groups, got %d", len(groups1))
	}

	// Verify groups are sorted by percentage
	for i := 1; i < len(groups1); i++ {
		if groups1[i-1][0].DuplicatedPercentage() < groups1[i][0].DuplicatedPercentage() {
			t.Errorf("Expected groups to be sorted by percentage")
		}
	}

	// Get groups for folder2
	groups2 := checker.GetSimilarityFolderGroup("folder2")
	if len(groups2) != 1 {
		t.Errorf("Expected folder2 to have 1 similarity group, got %d", len(groups2))
	}

	// Get groups for folder3
	groups3 := checker.GetSimilarityFolderGroup("folder3")
	if len(groups3) != 1 {
		t.Errorf("Expected folder3 to have 1 similarity group, got %d", len(groups3))
	}
}

func TestSimilarityChecker_ContainsSimilarityGroup(t *testing.T) {
	storage := NewMemoryStorageWithDupTrackingSkipRules([]string{})

	// Add files with duplicates
	storage.AddFile(&File{Path: "folder1/file1.txt", Hash: "hash1", Name: "file1.txt", Size: 1024, ModTime: time.Now()})
	storage.AddFile(&File{Path: "folder2/file1.txt", Hash: "hash1", Name: "file1.txt", Size: 1024, ModTime: time.Now()})
	storage.AddFile(&File{Path: "folder3/file2.txt", Hash: "hash2", Name: "file2.txt", Size: 2048, ModTime: time.Now()})
	storage.AddFile(&File{Path: "folder4/file2.txt", Hash: "hash2", Name: "file2.txt", Size: 2048, ModTime: time.Now()})

	checker := &SimilarityChecker{}
	err := checker.CalculateSimilarity(storage)
	if err != nil {
		t.Fatalf("CalculateSimilarity failed: %v", err)
	}

	// Test folders with similarities
	if !checker.ContainsSimilarityGroup("folder1") {
		t.Errorf("Expected folder1 to have similarity group")
	}
	if !checker.ContainsSimilarityGroup("folder2") {
		t.Errorf("Expected folder2 to have similarity group")
	}
	if !checker.ContainsSimilarityGroup("folder3") {
		t.Errorf("Expected folder3 to have similarity group")
	}
	if !checker.ContainsSimilarityGroup("folder4") {
		t.Errorf("Expected folder4 to have similarity group")
	}

	// Test folder without similarities
	if checker.ContainsSimilarityGroup("nonexistent") {
		t.Errorf("Expected nonexistent folder to not have similarity group")
	}
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		path     string
		expected []string
	}{
		{"folder1/folder2/file.txt", []string{"folder1", "folder2", "file.txt"}},
		{"file.txt", []string{"file.txt"}},
		{"folder1/folder2/folder3", []string{"folder1", "folder2", "folder3"}},
		{"a/b/c/d/e", []string{"a", "b", "c", "d", "e"}},
	}

	for _, tt := range tests {
		result := FileNameSplitByPath(tt.path)
		if len(result) != len(tt.expected) {
			t.Errorf("SplitPath(%s) length = %d, expected %d", tt.path, len(result), len(tt.expected))
			continue
		}

		for i, expected := range tt.expected {
			if result[i] != expected {
				t.Errorf("SplitPath(%s)[%d] = %s, expected %s", tt.path, i, result[i], expected)
			}
		}
	}
}

func TestSplitPath_Root(t *testing.T) {
	tests := []struct {
		path     string
		expected []string
	}{
		{".", []string{"."}},
		{"/", []string{"/"}},
		{"", []string{"."}},
	}

	for _, tt := range tests {
		result := FileNameSplitByPath(tt.path)
		if len(result) != len(tt.expected) {
			t.Errorf("SplitPath(%s) length = %d, expected %d", tt.path, len(result), len(tt.expected))
			continue
		}

		for i, expected := range tt.expected {
			if result[i] != expected {
				t.Errorf("SplitPath(%s)[%d] = %s, expected %s", tt.path, i, result[i], expected)
			}
		}
	}
}

func TestSimilarityChecker_CalculateSimilarity_EmptyStorage(t *testing.T) {
	storage := NewMemoryStorageWithDupTrackingSkipRules([]string{})

	checker := &SimilarityChecker{}
	err := checker.CalculateSimilarity(storage)
	if err != nil {
		t.Fatalf("CalculateSimilarity failed: %v", err)
	}

	// Should have no similarity groups
	similarityFolders := checker.GetSimilarityFolder()
	if len(similarityFolders) != 0 {
		t.Errorf("Expected no similarity folders for empty storage, got %d", len(similarityFolders))
	}

	if checker.ContainsSimilarityGroup("any") {
		t.Errorf("Expected no similarity groups for empty storage")
	}
}

func TestSimilarityChecker_CalculateSimilarity_SingleFile(t *testing.T) {
	storage := NewMemoryStorageWithDupTrackingSkipRules([]string{})

	// Add single file
	storage.AddFile(&File{Path: "folder/file.txt", Hash: "hash1", Name: "file.txt", Size: 1024, ModTime: time.Now()})

	checker := &SimilarityChecker{}
	err := checker.CalculateSimilarity(storage)
	if err != nil {
		t.Fatalf("CalculateSimilarity failed: %v", err)
	}

	// Should have no similarity groups (single file can't be duplicate)
	similarityFolders := checker.GetSimilarityFolder()
	if len(similarityFolders) != 0 {
		t.Errorf("Expected no similarity folders for single file, got %d", len(similarityFolders))
	}
}

func TestSimilarityChecker_CalculateSimilarity_SameFolderDuplicates(t *testing.T) {
	storage := NewMemoryStorageWithDupTrackingSkipRules([]string{})

	// Add files with same hash in same folder
	storage.AddFile(&File{Path: "folder/file1.txt", Hash: "hash1", Name: "file1.txt", Size: 1024, ModTime: time.Now()})
	storage.AddFile(&File{Path: "folder/file2.txt", Hash: "hash1", Name: "file2.txt", Size: 1024, ModTime: time.Now()})

	checker := &SimilarityChecker{}
	err := checker.CalculateSimilarity(storage)
	if err != nil {
		t.Fatalf("CalculateSimilarity failed: %v", err)
	}

	// Should have no similarity groups (same folder duplicates are skipped)
	similarityFolders := checker.GetSimilarityFolder()
	if len(similarityFolders) != 0 {
		t.Errorf("Expected no similarity folders for same folder duplicates, got %d", len(similarityFolders))
	}
}

func TestSimilarityChecker_GetDuplicatedFolderPair(t *testing.T) {
	folder1 := &Folder{Name: "folder1", Path: "folder1"}
	folder2 := &Folder{Name: "folder2", Path: "folder2"}
	folders := make(map[string][2]*FolderSimilarity)

	// Test first call
	fs1, fs2 := getDuplicatedFolderPair(folder1, folder2, folders)
	if fs1.Folder != folder1 || fs2.Folder != folder2 {
		t.Errorf("Expected correct folder assignment")
	}

	// Test second call with same folders
	fs1_2, fs2_2 := getDuplicatedFolderPair(folder1, folder2, folders)
	if fs1_2 != fs1 || fs2_2 != fs2 {
		t.Errorf("Expected same FolderSimilarity instances on second call")
	}

	// Test call with reversed order
	fs2_3, fs1_3 := getDuplicatedFolderPair(folder2, folder1, folders)
	if fs1_3 != fs1 || fs2_3 != fs2 {
		t.Errorf("Expected same FolderSimilarity instances with reversed order")
	}
}

func TestGetFolderSimilarity(t *testing.T) {
	folder1 := &Folder{Name: "folder1", Path: "folder1"}
	folder2 := &Folder{Name: "folder2", Path: "folder2"}
	folders := make(map[string][2]*FolderSimilarity)

	// Create folder pair
	fs1 := &FolderSimilarity{Folder: folder1}
	fs2 := &FolderSimilarity{Folder: folder2}
	folders["folder1:folder2"] = [2]*FolderSimilarity{fs1, fs2}

	// Test getting similarity
	result1, result2, err := getFolderSimilarity("folder1", "folder2", folders)
	if err != nil {
		t.Fatalf("getFolderSimilarity failed: %v", err)
	}

	if result1 != fs1 || result2 != fs2 {
		t.Errorf("Expected correct FolderSimilarity instances")
	}

	// Test with reversed order
	result2_2, result1_2, err := getFolderSimilarity("folder2", "folder1", folders)
	if err != nil {
		t.Fatalf("getFolderSimilarity failed with reversed order: %v", err)
	}

	if result1_2 != fs1 || result2_2 != fs2 {
		t.Errorf("Expected correct FolderSimilarity instances with reversed order")
	}

	// Test with non-existent pair
	_, _, err = getFolderSimilarity("nonexistent1", "nonexistent2", folders)
	if err == nil {
		t.Errorf("Expected error for non-existent folder pair")
	}
}
