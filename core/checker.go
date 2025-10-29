// Package core provides file similarity detection and folder comparison functionality.
// It includes storage management, file hashing, and duplicate detection algorithms.
package core

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// FolderSimilarity represents a folder with similarity analysis data.
type FolderSimilarity struct {
	*Folder
	FileCount          int
	DuplicateFileCount int
	TargetFolder       *FolderSimilarity
	DuplicateFiles     map[string]*File
}

// DuplicatedPercentage returns the percentage of duplicate files in this folder.
func (f *FolderSimilarity) DuplicatedPercentage() float64 {
	return float64(f.DuplicateFileCount) * 100.0 / float64(f.FileCount)
}

// SimilarityChecker analyzes folders and files to find duplicates and calculate similarity percentages.
type SimilarityChecker struct {
	similarityFolderPairs map[string][2]*FolderSimilarity
	similarityFolderMap   map[string][]string
}

func folderPairKey(path1 string, path2 string) string {
	if path1 > path2 {
		return fmt.Sprintf("%s:%s", path2, path1)
	}
	return fmt.Sprintf("%s:%s", path1, path2)
}

func (s *SimilarityChecker) getDuplicatedFolderPair(folder1 *Folder, folder2 *Folder, folders map[string][2]*FolderSimilarity) (*FolderSimilarity, *FolderSimilarity) {
	key := folderPairKey(folder1.Path, folder2.Path)

	pair, ok := folders[key]
	if !ok {
		pair = [2]*FolderSimilarity{
			{
				Folder:         folder1,
				FileCount:      folder1.GetFileCount(),
				DuplicateFiles: make(map[string]*File),
			},
			{
				Folder:         folder2,
				FileCount:      folder2.GetFileCount(),
				DuplicateFiles: make(map[string]*File),
			},
		}
		pair[0].TargetFolder = pair[1]
		pair[1].TargetFolder = pair[0]

		folders[key] = pair
	}

	if pair[0].Folder.Path == folder1.Path {
		return pair[0], pair[1]
	}
	return pair[1], pair[0]
}

func getFolderSimilarity(path1, path2 string, folders map[string][2]*FolderSimilarity) (*FolderSimilarity, *FolderSimilarity, error) {
	key := folderPairKey(path1, path2)
	pair, ok := folders[key]

	if !ok {
		return nil, nil, fmt.Errorf("folder pair not found")
	}
	if pair[0].Folder.Path == path1 {
		return pair[0], pair[1], nil
	}
	return pair[1], pair[0], nil
}

// CalculateSimilarity computes folder similarity based on duplicate files.
// This method should only be called once after all files are scanned.
// Calling it multiple times will produce incorrect results.
func (s *SimilarityChecker) CalculateSimilarity(storage Storage) error {
	matchedFiles, err := storage.GetMatchedFiles()
	if err != nil {
		return err
	}

	folders := make(map[string][2]*FolderSimilarity)

	// calculate file similarity
	for _, matchedFile := range matchedFiles {
		for i := 0; i < len(matchedFile.Files); i++ {
			for j := i + 1; j < len(matchedFile.Files); j++ {
				folder1, folder2 := s.getDuplicatedFolderPair(matchedFile.Files[i].Parent, matchedFile.Files[j].Parent, folders)

				if _, ok := folder1.DuplicateFiles[matchedFile.Files[i].Name]; !ok {
					folder1.DuplicateFiles[matchedFile.Files[i].Name] = matchedFile.Files[i]
					folder1.DuplicateFileCount++
				}
				if _, ok := folder2.DuplicateFiles[matchedFile.Files[j].Name]; !ok {
					folder2.DuplicateFiles[matchedFile.Files[j].Name] = matchedFile.Files[j]
					folder2.DuplicateFileCount++
				}
			}
		}
	}

	// apply matched folder count to parent folder
	parentFolders := maps.Clone(folders)
	for _, matchedFolders := range folders {
		folder1, folder2 := matchedFolders[0], matchedFolders[1]
		s.calculateParentFolderSimilarity(folder1, folder2, parentFolders)
	}

	s.similarityFolderPairs = parentFolders

	s.similarityFolderMap = make(map[string][]string)
	for key := range s.similarityFolderPairs {
		paths := strings.Split(key, ":")
		if len(paths) != 2 {
			// invalid key
			continue
		}

		if paths[0] == paths[1] {
			// Skip same-folder duplicates as they are handled separately
			// TODO: Implement GetSameFolderDuplicates() method
			continue
		} else {
			s.similarityFolderMap[paths[0]] = append(s.similarityFolderMap[paths[0]], key)
			s.similarityFolderMap[paths[1]] = append(s.similarityFolderMap[paths[1]], key)
		}
	}
	return nil
}

func (s *SimilarityChecker) ContainsSimilarityGroup(path string) bool {
	_, ok := s.similarityFolderMap[path]
	return ok
}

func (s *SimilarityChecker) GetSimilarityFolderGroup(path string) [][2]*FolderSimilarity {
	output := [][2]*FolderSimilarity{}

	for _, pairPath := range s.similarityFolderMap[path] {
		paths := strings.Split(pairPath, ":")
		if len(paths) != 2 {
			// invalid key
			continue
		}

		f1, f2, err := getFolderSimilarity(paths[0], paths[1], s.similarityFolderPairs)
		if err != nil {
			continue
		}
		if f1.Folder.Path == path {
			output = append(output, [2]*FolderSimilarity{f1, f2})
		} else {
			output = append(output, [2]*FolderSimilarity{f2, f1})
		}
	}

	// sort output by DuplicateFileCount
	sort.Slice(output, func(i, j int) bool {
		p1, p2 := output[i][0].DuplicatedPercentage(), output[j][0].DuplicatedPercentage()
		if p1 == p2 {
			p1, p2 := output[i][1].DuplicatedPercentage(), output[j][1].DuplicatedPercentage()
			if p1 == p2 {
				return len(output[i][0].DuplicateFiles) > len(output[j][0].DuplicateFiles)
			}
			return p1 > p2
		}
		return p1 > p2
	})

	// filteredOutput := [][2]*FolderSimilarity{}

	// a, b := float64(0), float64(0)
	// for _, pair := range output {
	// 	if pair[0].DuplicatedPercentage() != a || pair[1].DuplicatedPercentage() != b {
	// 		a = pair[0].DuplicatedPercentage()
	// 		b = pair[1].DuplicatedPercentage()
	// 		filteredOutput = append(filteredOutput, pair)
	// 	}
	// }

	return output
}

// calculateParentFolderSimilarity calculates the similarity between two parent folders
func (s *SimilarityChecker) calculateParentFolderSimilarity(
	folder1, folder2 *FolderSimilarity,
	folders map[string][2]*FolderSimilarity) {
	if folder1 == folder2 || folder1.Parent == folder2.Parent {
		return
	}

	parents := SplitPath(folder1.Path)
	parents2 := SplitPath(folder2.Path)

	// find the common parent of folder1 and folder2
	commonParent := "."
	for i := 0; i < len(parents) && i < len(parents2); i++ {
		if parents[i] != parents2[i] {
			break
		}
		commonParent = filepath.Join(commonParent, parents[i])
	}

	// loop all possible folder pair between folder1 and folder2
	currentFolder1 := folder1.Folder
	for currentFolder1.Path != commonParent {
		currentFolder2 := folder2.Folder
		for currentFolder2.Path != commonParent {
			f1, f2 := s.getDuplicatedFolderPair(currentFolder1, currentFolder2, folders)
			if f1 != folder1 {
				f1.DuplicateFileCount += folder1.DuplicateFileCount
			}
			if f2 != folder2 {
				f2.DuplicateFileCount += folder2.DuplicateFileCount
			}

			currentFolder2 = currentFolder2.Parent
		}
		currentFolder1 = currentFolder1.Parent
	}
}

func (s *SimilarityChecker) GetSimilarityFolder() []string {
	output := make([]string, len(s.similarityFolderMap))
	i := 0
	for path := range s.similarityFolderMap {
		output[i] = path
		i++
	}
	return output
}

func SplitPath(path string) []string {
	output := []string{filepath.Base(path)}
	for path != "." && path != "/" {
		folder := filepath.Base(filepath.Dir(path))
		if folder != "." && folder != "/" {
			output = append([]string{folder}, output...)
		}
		path = filepath.Dir(path)
	}
	return output
}

// return the child folder match with folder2 child
func (s *SimilarityChecker) GetChildFolderSimilarityMatch(f1, f2 *FolderSimilarity) (matchedPairs [][2]*FolderSimilarity, folder1Only []*Folder, folder2Only []*Folder) {
	folder2Child := map[string]*Folder{}

	for _, f := range f2.GetFolders() {
		folder2Child[f.Path] = f
	}

	for _, f := range f1.GetFolders() {
		groups := s.GetSimilarityFolderGroup(f.Path)

		matched := false
		for _, group := range groups {
			if _, ok := folder2Child[group[1].Folder.Path]; ok {
				matchedPairs = append(matchedPairs, group)
				matched = true
				delete(folder2Child, group[1].Folder.Path)
				break
			}
		}

		if !matched {
			folder1Only = append(folder1Only, f)
		}
	}
	for _, f := range folder2Child {
		folder2Only = append(folder2Only, f)
	}

	return matchedPairs, folder1Only, folder2Only
}

// helper function to filter file folder not within the map
func FolderNotInMap(folders []*Folder, folderMap [][2]*FolderSimilarity) []*Folder {
	childFolders := map[string]*Folder{}
	for _, folder := range folders {
		childFolders[folder.Path] = folder
	}
	for _, folder := range folderMap {
		if _, ok := childFolders[folder[0].Folder.Path]; ok {
			delete(childFolders, folder[0].Folder.Path)
		}
		if _, ok := childFolders[folder[1].Folder.Path]; ok {
			delete(childFolders, folder[1].Folder.Path)
		}
	}
	output := make([]*Folder, len(childFolders))
	i := 0
	for _, folder := range childFolders {
		output[i] = folder
		i++
	}
	return output
}

func (s *SimilarityChecker) DeleteSimilarityGroup(folder1, folder2 *FolderSimilarity) {
	parents := SplitPath(folder1.Path)
	parents2 := SplitPath(folder2.Path)

	// find the common parent of folder1 and folder2
	commonParent := "."
	for i := 0; i < len(parents) && i < len(parents2); i++ {
		if parents[i] != parents2[i] {
			break
		}
		commonParent = filepath.Join(commonParent, parents[i])
	}

	f1DuplicateFileCount := folder1.DuplicateFileCount
	f2DuplicateFileCount := folder2.DuplicateFileCount

	deletedKeys := []string{}

	// loop all possible folder pair between folder1 and folder2
	currentFolder1 := folder1.Folder
	for currentFolder1.Path != commonParent {
		currentFolder2 := folder2.Folder
		for currentFolder2.Path != commonParent {
			key := folderPairKey(currentFolder1.Path, currentFolder2.Path)
			if _, ok := s.similarityFolderPairs[key]; ok {

				f1, f2 := s.getDuplicatedFolderPair(currentFolder1, currentFolder2, s.similarityFolderPairs)
				// if f1 != folder1 {
				f1.DuplicateFileCount -= f1DuplicateFileCount
				// }
				// if f2 != folder2 {
				f2.DuplicateFileCount -= f2DuplicateFileCount
				// }

				if f2.DuplicateFileCount == 0 || f1.DuplicateFileCount == 0 {
					delete(s.similarityFolderPairs, key)
					deletedKeys = append(deletedKeys, key)
				}
			}

			currentFolder2 = currentFolder2.Parent
		}
		currentFolder1 = currentFolder1.Parent
	}

	// delete the key map
	for _, key := range deletedKeys {
		folders := strings.Split(key, ":")
		for _, folder := range folders {
			s.similarityFolderMap[folder] = slices.DeleteFunc(s.similarityFolderMap[folder], func(k string) bool {
				return k == key
			})
			if len(s.similarityFolderMap[folder]) == 0 {
				delete(s.similarityFolderMap, folder)
			}
		}
	}
}

// Helper function to get the matched file pairs
func GetMatchedFilePairs(folder1, folder2 *FolderSimilarity) (matchedPairs [][2]*File, folder1Only []*File, folder2Only []*File) {
	files1 := folder1.GetFiles()
	files2 := folder2.GetFiles()

	sort.Slice(files1, func(i, j int) bool {
		return files1[i].Hash < files1[j].Hash
	})
	sort.Slice(files2, func(i, j int) bool {
		return files2[i].Hash < files2[j].Hash
	})

	a, b := 0, 0
	for a < len(files1) || b < len(files2) {
		if a >= len(files1) {
			folder2Only = append(folder2Only, files2[b])
			b++
		} else if b >= len(files2) {
			folder1Only = append(folder1Only, files1[a])
			a++
		} else if files1[a].Hash == files2[b].Hash {
			matchedPairs = append(matchedPairs, [2]*File{files1[a], files2[b]})
			a++
			b++
		} else if files1[a].Hash < files2[b].Hash {
			folder1Only = append(folder1Only, files1[a])
			a++
		} else {
			folder2Only = append(folder2Only, files2[b])
			b++
		}
	}

	return matchedPairs, folder1Only, folder2Only
}

func (s *SimilarityChecker) GenerateMergeFolderPair(folder1, folder2 *FolderSimilarity) MergeFolderPair {
	p := MergeFolderPair{
		Folder1:   folder1,
		Folder2:   folder2,
		MatchType: MatchBothSide,

		FilePairs:   []MergeFilePair{},
		FolderPairs: []MergeFolderPair{},
	}

	matchedPairs, f1Files, f2Files := GetMatchedFilePairs(folder1, folder2)

	for _, pair := range matchedPairs {
		p.FilePairs = append(p.FilePairs, MergeFilePair{File1: pair[0], File2: pair[1]})
	}
	for _, file := range f1Files {
		p.FilePairs = append(p.FilePairs, MergeFilePair{File1: file, File2: nil})
	}
	for _, file := range f2Files {
		p.FilePairs = append(p.FilePairs, MergeFilePair{File1: nil, File2: file})
	}

	matchedSubFolders, f1Folders, f2Folders := s.GetChildFolderSimilarityMatch(folder1, folder2)
	for _, pair := range matchedSubFolders {
		// p.folderPairs = append(p.folderPairs, MergeFolderPair{Folder1: pair[0], Folder2: pair[1], MatchType: MatchBothSide})
		p.FolderPairs = append(p.FolderPairs, s.GenerateMergeFolderPair(pair[0], pair[1]))
	}
	for _, f1only := range f1Folders {
		p.FolderPairs = append(p.FolderPairs, MergeFolderPair{Folder1: f1only, Folder2: nil, MatchType: MatchOnlyLeft})
	}
	for _, f2only := range f2Folders {
		p.FolderPairs = append(p.FolderPairs, MergeFolderPair{Folder1: nil, Folder2: f2only, MatchType: MatchOnlyRight})
	}
	return p
}

// TODO: delete it later
func (s *SimilarityChecker) DebugPringInfo() {
	fmt.Println("Debug Print Info")
	for k, _ := range s.similarityFolderPairs {
		fmt.Println(k)
	}
	fmt.Println("--------------------------------")
	for k, _ := range s.similarityFolderMap {
		fmt.Println(k, "->", strings.Join(s.similarityFolderMap[k], ","))
	}
	fmt.Println("--------------------------------")
}
