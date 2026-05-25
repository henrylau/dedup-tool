# Core package review

Focused on bugs, concurrency, and correctness in `core/`. UI excluded.

## Bugs (likely actual problems)

**`scanner.go:41` — empty filename panics, hidden dirs not skipped**
- `d.Name()[0] == '.'` panics on empty name.
- Returning `nil` for a hidden directory still descends into it. Use `fs.SkipDir` when `d.IsDir()`.

**`scanner.go:49` — fd leak under `WalkDir`**
- `defer f.Close()` inside the walk callback only fires when `WalkDir` returns, so every file scanned stays open until the whole tree is done. Wrap each file in a function or call `f.Close()` explicitly before returning.

**`action.go:118` — `DeleteFolder` cannot delete non-empty folder**
- Uses `root.Remove`, which fails on a populated directory. The action is generated for `MatchOnlyLeft/Right` (a whole folder the user wants gone), so it should be `RemoveAll`. Today the action silently fails for any folder with contents.

**`action.go:60` — Move stale storage**
- When `exists` is true at the destination, the source `File` is removed from storage but the destination is never registered. Storage drifts from disk. Even when `!exists`, the new `File` carries `task.File.Hash` without re-confirming, which is fine but worth a comment.

**`storage.go:96` — `GetFolder` race (TOCTOU)**
- `Load` then `Store`: two goroutines hitting the same path produce two `*Folder` instances; only one wins in `s.folders`, but the other is still attached to a parent via `parentFolder.Folders.Store`. Use `LoadOrStore` and discard the loser. (Today scans are single-goroutine, but `Storage` is documented for concurrent file adds via `sync.Map`, so the contract is broken.)

**`storage.go:39` — racy slice mutation in `MatchedFileGroup.Files`**
- `slices.DeleteFunc` and `append` on `pair.Files` aren't synchronized; concurrent `AddFile`/`RemoveFile` for the same hash will tear the slice. The hash bucket needs its own mutex.

**`type.go:67` — `fileCountCache` uses `0` as "invalid" sentinel**
- An empty folder caches `0`, which then forces a full recursive recount on every call. Use a separate `valid` flag or `int32(-1)` sentinel.
- Also racy: `GetFileCount` does load → recurse → store with no lock; an `AddFile` that lands between the recurse and the store invalidates first, then `GetFileCount` overwrites with stale value. Cache will silently skew.

**`checker.go:25` — divide-by-zero**
- `DuplicatedPercentage` divides by `FileCount`. Empty folder → `NaN`/`+Inf`. Guard with `if f.FileCount == 0 { return 0 }`.

**`executor.go:85` — unconditional `time.Sleep(10ms)` in production**
- Drops throughput by 100/sec for no apparent reason. If it's there to debounce UI updates, it belongs in the UI consumer, not the core executor.

**`executor.go:52` — `progressChan` never closed; `done` field never read**
- Receivers can't tell when the run is finished without inspecting context. Either close the channel in a `defer` or remove the field.

## API / design smells

- `Folder.AddFile`/`RemoveFile` return `error` but never produce one — interface noise.
- `MergeFolderPair.Folder1/Folder2` are `interface{}` discriminated by `MatchType`. Every method is a chain of type assertions. A small interface (`MergeNode { GetName, GetFileCount, ... }`) or two distinct types would make the invariants typesafe.
- `core.Scanner.Logger func(string)` vs `core.Logger` interface elsewhere — pick one.
- `MemoryStorage.ExportStorage` is exported but `Storage` interface doesn't include it. Either add it to the interface or move it to a concrete-only helper. Also `for i, _ := range files` should be `for i := range files`.
- `getDuplicatedFolderPair` is called with a writable map for both reads and creates; `getFolderSimilarity` is read-only. Splitting "lookup" from "lookup-or-create" would prevent accidental mutation in read paths (as `GetSimilarityFolderGroup` does today).
- `ContainsSimilarityGroup` does a linear scan over the whole map for descendant lookups (`TODO: optimize`). For deep trees, build an index keyed by every ancestor path during `CalculateSimilarity`, or store paths in a sorted slice for prefix-binary-search.
- `CalculateSimilarity` doc says "only call once" — enforce it (panic / return error on second call) instead of relying on the comment.
- `parentFolders := maps.Clone(folders)` is a shallow clone; the `[2]*FolderSimilarity` arrays are copied by value but the pointed-to `FolderSimilarity`s are shared. The current code happens to be correct because `calculateParentFolderSimilarity` only mutates `DuplicateFileCount` on the *innermost* targets via `getDuplicatedFolderPair` against `parentFolders`, but the aliasing is subtle — a comment would help future readers.

## Smaller nits

- `checker.go:30 folderPairKey` uses `:` as separator — fine on POSIX, but Windows paths can contain `:` (drive letters). If Windows is in scope, pick a separator that can't collide and `strings.SplitN(key, sep, 2)`.
- `FileNameSplitByPath` rebuilds the prefix by repeatedly prepending — `O(n²)`. `strings.Split(filepath.ToSlash(path), "/")` after trimming is `O(n)`.
- `storage.go:49` returns an error after the inconsistent-state branch is already taken; either fix invariants first or treat as `panic` since the comment says "should not happen".
- `scanner.go:82-126` — large commented-out `ScanFolder`. Delete; git history is the archive.
- `core/scanner_test.go` is essentially empty (1 line). Either populate or remove.
- `helper.go:15 getFileHash` requires `io.ReaderAt`. `os.Root.Open` returns `*os.File` which satisfies it, but the assertion against `fs.File` is fragile if `Scanner` ever swaps in a different FS.

## Concurrency summary

The package presents itself as concurrency-safe (sync.Map everywhere, atomics for counters), but there are several non-atomic read-modify-write sequences that defeat that goal: `GetFolder` create path, `MatchedFileGroup.Files` mutation, `fileCountCache` recompute, and `invalidateCache` walking up the tree without ordering against the count update. If concurrent scanning is a real goal (Scanner has `Path []string`), each of these needs a lock per folder/per hash bucket.

## Suggested priorities

1. The `defer f.Close()` fd leak (scanner) and the `DeleteFolder` non-empty-fail (action) — both are silent-correctness bugs.
2. `GetFolder` `LoadOrStore` and per-bucket locking for `MatchedFileGroup.Files` if any concurrent scanning is planned.
3. The 10ms sleep in the executor — easy win, unclear why it's there.
4. The `interface{}` typing in `MergeFolderPair` — biggest readability cost in the package.
