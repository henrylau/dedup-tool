package service

import (
	"os"
	"path/filepath"
	"testing"

	"folder-similarity/app/dto"
)

func TestListDir_RejectsBadInputs(t *testing.T) {
	s := &Similarity{}

	if _, err := s.ListDir("", ""); err == nil {
		t.Fatal("expected error for empty path")
	}
	if _, err := s.ListDir("relative/path", ""); err == nil {
		t.Fatal("expected error for relative path")
	}
	if _, err := s.ListDir("/__definitely_not_a_real_dir__xyz", ""); err == nil {
		t.Fatal("expected error for nonexistent dir")
	}
}

func TestListDir_FiltersAndSorts(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "zeta"))
	mustMkdir(t, filepath.Join(root, "alpha"))
	mustMkdir(t, filepath.Join(root, ".hidden_dir"))
	mustWrite(t, filepath.Join(root, "data.json"))
	mustWrite(t, filepath.Join(root, "data.txt"))
	mustWrite(t, filepath.Join(root, ".hidden.json"))

	s := &Similarity{}

	// No filter: only visible directories.
	out, err := s.ListDir(root, "")
	if err != nil {
		t.Fatalf("ListDir(no filter): %v", err)
	}
	if got, want := names(out), []string{"alpha", "zeta"}; !equalStrings(got, want) {
		t.Fatalf("no-filter names: got %v, want %v", got, want)
	}

	// With .json filter: dirs first (sorted), then matching files (sorted).
	out, err = s.ListDir(root, ".json")
	if err != nil {
		t.Fatalf("ListDir(.json): %v", err)
	}
	if got, want := names(out), []string{"alpha", "zeta", "data.json"}; !equalStrings(got, want) {
		t.Fatalf("json-filter names: got %v, want %v", got, want)
	}
	// Filter is case-insensitive on extension.
	if out2, err := s.ListDir(root, ".JSON"); err != nil || !equalStrings(names(out2), names(out)) {
		t.Fatalf("expected case-insensitive ext filter, got %v err=%v", names(out2), err)
	}
}

func TestListDir_CleanResolvesTraversal(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "child"))

	s := &Similarity{}
	// filepath.Clean collapses "..": root/child/.. == root.
	out, err := s.ListDir(filepath.Join(root, "child", ".."), "")
	if err != nil {
		t.Fatalf("ListDir(traversal): %v", err)
	}
	if got, want := names(out), []string{"child"}; !equalStrings(got, want) {
		t.Fatalf("traversal-resolve names: got %v, want %v", got, want)
	}
}

func mustMkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.Mkdir(p, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", p, err)
	}
}

func mustWrite(t *testing.T, p string) {
	t.Helper()
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
}

func names(es []dto.DirEntry) []string {
	out := make([]string, len(es))
	for i, e := range es {
		out[i] = e.Name
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
