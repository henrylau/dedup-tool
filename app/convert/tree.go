package convert

import (
	"folder-similarity/app/dto"
	"folder-similarity/core"
	"sort"
)

// TreeFromFolder returns a DTO tree from a storage folder.
func TreeFromFolder(root *core.Folder) dto.FolderNode {
	if root == nil {
		return dto.FolderNode{}
	}
	return buildNode(root)
}

// TreeFromFolderPrune returns a tree that optionally keeps only branches that contain
// similarity (including descendants), matching the TUI filter predicate.
func TreeFromFolderPrune(root *core.Folder, keep func(path string) bool) dto.FolderNode {
	if root == nil {
		return dto.FolderNode{}
	}
	if keep == nil {
		return buildNode(root)
	}
	return buildNodePrune(root, keep)
}

func buildNode(f *core.Folder) dto.FolderNode {
	folders := f.GetFolders()
	sort.Slice(folders, func(i, j int) bool {
		return folders[i].Name < folders[j].Name
	})
	children := make([]dto.FolderNode, 0, len(folders))
	for _, sub := range folders {
		children = append(children, buildNode(sub))
	}
	return dto.FolderNode{
		Path:           f.Path,
		Name:           f.Name,
		TotalSizeBytes: f.GetTotalFilesSizeRecursive(),
		Children:       children,
	}
}

func buildNodePrune(f *core.Folder, keep func(string) bool) dto.FolderNode {
	folders := f.GetFolders()
	sort.Slice(folders, func(i, j int) bool {
		return folders[i].Name < folders[j].Name
	})
	children := make([]dto.FolderNode, 0, len(folders))
	for _, sub := range folders {
		if n := buildNodePrune(sub, keep); n.Path != "" {
			children = append(children, n)
		}
	}
	selfKeep := keep(f.Path)
	if !selfKeep && len(children) == 0 {
		return dto.FolderNode{}
	}
	return dto.FolderNode{
		Path:           f.Path,
		Name:           f.Name,
		TotalSizeBytes: f.GetTotalFilesSizeRecursive(),
		Children:       children,
	}
}
