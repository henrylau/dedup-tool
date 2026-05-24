package convert

import (
	"folder-similarity/core"
	"testing"
)

func TestTreeFromFolder(t *testing.T) {
	r := &core.Folder{Name: ".", Path: "."}
	a := &core.Folder{Name: "a", Path: "a", Parent: r}
	r.Folders.Store("a", a)
	n := TreeFromFolder(r)
	if n.Path != "." {
		t.Fatalf("root path: %q", n.Path)
	}
	if len(n.Children) != 1 || n.Children[0].Path != "a" {
		t.Fatalf("children: %+v", n.Children)
	}
}
