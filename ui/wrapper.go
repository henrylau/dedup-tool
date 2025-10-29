package ui

import (
	"folder-similarity/core"
	"folder-similarity/ui/tree"
)

// FolderItemWrapper wraps core.Folder to implement tree.Item interface
type FolderItemWrapper struct {
	*core.Folder
	childrenItem []tree.Item
	parentItem   tree.Item
}

// GetChildren implements tree.Item.
func (f *FolderItemWrapper) GetChildren() []tree.Item {
	if f.childrenItem != nil {
		return f.childrenItem
	}

	for _, folder := range f.GetFolders() {
		f.childrenItem = append(f.childrenItem, &FolderItemWrapper{Folder: folder, parentItem: f})
	}
	return f.childrenItem
}

// GetName implements tree.Item.
func (f *FolderItemWrapper) GetName() string {
	return f.Name
}

// Parent implements tree.Item.
func (f *FolderItemWrapper) Parent() tree.Item {
	if f.Folder.Parent == nil {
		return nil
	}
	if f.parentItem != nil {
		return f.parentItem
	}

	return &FolderItemWrapper{Folder: f.Folder.Parent, parentItem: f}
}

var _ tree.Item = &FolderItemWrapper{}
