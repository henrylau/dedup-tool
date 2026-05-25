package pathutil

import (
	"path/filepath"
	"strings"
)

// IsRootOrUnder reports whether p is the root or a strict subpath of root, using OS-aware cleaning.
// Both paths should be clean absolute paths, or p may be "" (false).
func IsRootOrUnder(root, p string) bool {
	if p == "" {
		return false
	}
	root = filepath.Clean(root)
	p = filepath.Clean(p)
	if p == root {
		return true
	}
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// RelToRoot returns path relative to root as slash-separated segments like storage paths ("." for root of scan).
// root and abs must be absolute, clean paths. abs should be a path under root.
func RelToRoot(root, abs string) (string, error) {
	root = filepath.Clean(root)
	abs = filepath.Clean(abs)
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return "", err
	}
	if rel == "." {
		return ".", nil
	}
	return filepath.ToSlash(rel), nil
}

// NormalizeRelStorageKey converts a relative key from the tree (slash form) to OS form for filepath ops.
func NormalizeRelStorageKey(rel string) string {
	if rel == "" || rel == "." {
		return "."
	}
	return filepath.FromSlash(rel)
}
