package main

import (
	"encoding/json"
	"fmt"
	"folder-similarity/core"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSplitPath(t *testing.T) {
	path := "/tmp/cache/00/00/0000649b-baa5-4ce0-8cdb/-f84b155b37cb_thumb.webp"

	fmt.Println(parentFolders(path))
}

func parentFolders(path string) []string {
	output := []string{}
	for path != "." && path != "/" {
		folder := filepath.Base(filepath.Dir(path))
		if folder != "." && folder != "/" {
			output = append([]string{folder}, output...)
		}
		path = filepath.Dir(path)
	}

	return output
}

func ScanFiles(root []string, workerCount int) {
}

func TestMemoryStorage(t *testing.T) {
	storage := core.NewMemoryStorage()
	checker := core.SimilarityChecker{}
	jsonDataBytes, err := os.ReadFile("jsonData.json")
	if err != nil {
		t.Fatalf("Failed to read json data: %v", err)
	}
	var files []core.File
	err = json.Unmarshal(jsonDataBytes, &files)
	if err != nil {
		t.Fatalf("Failed to unmarshal json data: %v", err)
	}
	for _, file := range files {
		storage.AddFile(&file)
	}

	checker.CalculateSimilarity(storage)

	groups := checker.GetSimilarityFolderGroup("d")

	// checker.DebugPringInfo()

	var mergeFolderPair core.MergeFolderPair

	for _, group := range groups {
		if group[1].Folder.Path != "a" {
			continue
		}
		fmt.Printf("%s-%s: %d/%d %d/%d\n", group[0].Folder.Path, group[1].Folder.Path, group[0].DuplicateFileCount, group[0].FileCount, group[1].DuplicateFileCount, group[1].FileCount)

		// f1 := group[0].Folder
		// f2 := group[1].Folder

		// matches := checker.GeChildFolderSimilarityMatch(group[0], group[1])
		// for _, group := range matches {
		// 	fmt.Printf("%s-%s: %d/%d %d/%d\n", group[0].Folder.Path, group[1].Folder.Path, group[0].DuplicateFileCount, group[0].FileCount, group[1].DuplicateFileCount, group[1].FileCount)
		// }
		// for _, f1only := range core.FolderNotInMap(f1.GetFolders(), matches) {
		// 	fmt.Printf("f1 only: %s, file count: %d\n", f1only.Path, f1only.GetFileCount())
		// }
		// for _, f2only := range core.FolderNotInMap(f2.GetFolders(), matches) {
		// 	fmt.Printf("f2 only: %s, file count: %d\n", f2only.Path, f2only.GetFileCount())
		// }
		mergeFolderPair = checker.GenerateMergeFolderPair(group[0], group[1])
		break
	}

	for _, pair := range mergeFolderPair.FilePairs {
		fmt.Printf("%s - %s\n", pair.GetName(0), pair.GetName(1))
	}
	fmt.Println("--------------------------------")
	for _, pair := range mergeFolderPair.FolderPairs {
		fmt.Printf("%s - %s\n", pair.GetName(0), pair.GetName(1))
	}
}

func TestOpenFinder(t *testing.T) {
	err := exec.Command("open", "../photoview/assets").Run()
	if err != nil {
		t.Fatalf("Failed to open finder: %v", err)
	}
}

func TestListFolder(t *testing.T) {
	root, err := os.OpenRoot("../photoview/assets")
	if err != nil {
		t.Fatalf("Failed to open root: %v", err)
	}
	defer root.Close()

	path := "d/1"
	// Remove folder if it is empty (even if it only contains hidden files like .DS_Store)
	dir, err := root.Open(path)
	if err != nil {
		t.Fatalf("Failed to open dir: %v", err)
	}
	defer dir.Close()

	entries, err := dir.Readdir(0)
	if err != nil {
		t.Fatalf("Failed to read dir: %v", err)
	}

	remove := true
	for _, entry := range entries {
		// If entry is not a hidden file, treat folder as non-empty
		name := entry.Name()
		if name == "." || name == ".." {
			continue
		}
		if !strings.HasPrefix(name, ".") {
			remove = false
			break
		}
	}

	if remove {
		err := root.Remove(path)
		if err != nil {
			t.Fatalf("Failed to remove folder: %v", err)
		}
	}
}

func TestLoadData(t *testing.T) {

	storage := core.NewMemoryStorage()
	checker := core.SimilarityChecker{}
	jsonDataBytes, err := os.ReadFile("db 2.json")
	if err != nil {
		t.Fatalf("Failed to read json data: %v", err)
	}
	var files []core.File
	err = json.Unmarshal(jsonDataBytes, &files)
	if err != nil {
		t.Fatalf("Failed to unmarshal json data: %v", err)
	}
	for _, file := range files {
		storage.AddFile(&file)
	}

	checker.CalculateSimilarity(storage)

	checker.ContainsSimilarityGroup("寫真")
}
