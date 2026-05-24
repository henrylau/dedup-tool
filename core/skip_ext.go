package core

import "strings"

// DefaultSkipExtensions are skipped for directory scans (Scanner) and duplicate/hash indexing (MemoryStorage)
// when rule slices are nil.
var DefaultSkipExtensions = []string{".txt", ".lnk"}

// NormalizeSkipExtension returns a lower-case extension with a leading dot, or "" if empty after trim.
func NormalizeSkipExtension(ext string) string {
	ext = strings.TrimSpace(strings.ToLower(ext))
	if ext == "" {
		return ""
	}
	if ext[0] != '.' {
		ext = "." + ext
	}
	return ext
}

// NormalizeSkipExtensions normalizes each entry and drops empties.
func NormalizeSkipExtensions(exts []string) []string {
	if len(exts) == 0 {
		return nil
	}
	out := make([]string, 0, len(exts))
	for _, e := range exts {
		if n := NormalizeSkipExtension(e); n != "" {
			out = append(out, n)
		}
	}
	return out
}

// NormalizedSkipExtensionSet builds a lookup set for extension-based skips (O(1) by ext).
// ruleSlice nil uses DefaultSkipExtensions; a non-nil empty slice means nothing is skipped (returns nil).
func NormalizedSkipExtensionSet(ruleSlice []string) map[string]struct{} {
	var list []string
	switch {
	case ruleSlice == nil:
		list = DefaultSkipExtensions
	case len(ruleSlice) == 0:
		return nil
	default:
		list = NormalizeSkipExtensions(ruleSlice)
	}
	if len(list) == 0 {
		return nil
	}
	m := make(map[string]struct{}, len(list))
	for _, e := range list {
		m[e] = struct{}{}
	}
	return m
}

// ExtensionMatchesSkipSet reports whether ext (e.g. filepath.Ext(file.Name)) is in the skip set.
func ExtensionMatchesSkipSet(ext string, skip map[string]struct{}) bool {
	if skip == nil || ext == "" {
		return false
	}
	_, ok := skip[strings.ToLower(ext)]
	return ok
}
