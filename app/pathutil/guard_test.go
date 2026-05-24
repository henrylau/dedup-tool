package pathutil

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestIsRootOrUnder(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("path tests use Unix-style temp paths")
	}
	root := "/tmp/proj"
	if !IsRootOrUnder(root, "/tmp/proj") {
		t.Fatal("root should be under self")
	}
	if !IsRootOrUnder(root, "/tmp/proj/sub") {
		t.Fatal("sub should be under")
	}
	if IsRootOrUnder(root, "/tmp/other") {
		t.Fatal("sibling not under")
	}
	if IsRootOrUnder(root, "/tmp/proj/../other") {
		t.Fatal("escaped path not under")
	}
}

func TestRelToRoot(t *testing.T) {
	root := "/tmp/p"
	abs := filepath.Join(root, "a", "b")
	s, err := RelToRoot(root, abs)
	if err != nil {
		t.Fatal(err)
	}
	if s != "a/b" {
		t.Fatalf("got %q", s)
	}
	s, err = RelToRoot(root, root)
	if err != nil {
		t.Fatal(err)
	}
	if s != "." {
		t.Fatalf("got %q", s)
	}
}
