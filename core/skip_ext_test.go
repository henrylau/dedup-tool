package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeSkipExtension(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"txt", ".txt"},
		{".TXT", ".txt"},
		{" .lnk ", ".lnk"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := NormalizeSkipExtension(tt.in); got != tt.want {
			t.Errorf("NormalizeSkipExtension(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestNormalizedSkipExtensionSet_nilUsesDefault(t *testing.T) {
	m := NormalizedSkipExtensionSet(nil)
	if len(m) != 2 {
		t.Fatalf("want 2 defaults, got %v", m)
	}
	if _, ok := m[".txt"]; !ok {
		t.Fatal("missing .txt")
	}
	if _, ok := m[".lnk"]; !ok {
		t.Fatal("missing .lnk")
	}
}

func TestNormalizedSkipExtensionSet_emptyMeansNoSkip(t *testing.T) {
	m := NormalizedSkipExtensionSet([]string{})
	if m != nil {
		t.Fatalf("want nil map, got %v", m)
	}
}

func TestScannerSkipExtensionsBeforeAddFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "x.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "y.go"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	st := NewMemoryStorageWithDupTrackingSkipRules([]string{})
	sc := Scanner{Path: []string{dir}, Storage: st}
	if err := sc.Scan(); err != nil {
		t.Fatal(err)
	}
	root, err := st.GetFolder(".")
	if err != nil {
		t.Fatal(err)
	}
	if len(root.GetFiles()) != 1 || root.GetFiles()[0].Name != "y.go" {
		t.Fatalf("scanner should skip .txt by default, got %+v", root.GetFiles())
	}
}
