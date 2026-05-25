// Package dto holds JSON-friendly types for the Wails frontend.
package dto

// Phase describes high-level app lifecycle for the UI.
const (
	PhaseIdle     = "idle"
	PhaseScanning = "scanning"
	PhaseLoading  = "loading"
	PhaseReady    = "ready"
	PhaseError    = "error"
)

// PhaseEvent is emitted when scan/load state changes.
type PhaseEvent struct {
	Phase   string `json:"phase"`
	Message string `json:"message,omitempty"`
}

// ScanProgressEvent is emitted while hashing files during StartScan (throttled in the service layer).
type ScanProgressEvent struct {
	FilesScanned int `json:"filesScanned"`
}

// AppState is returned by GetState.
type AppState struct {
	Phase     string `json:"phase"`
	Message   string `json:"message,omitempty"`
	RootPath  string `json:"rootPath,omitempty"`
	HasData   bool   `json:"hasData"`
	Selected  string `json:"selected,omitempty"`
	Scanning  bool   `json:"scanning"`
	// Similarity group picker (phase 4): GroupIndex is -1 until a group is chosen when GroupCount > 1.
	GroupCount    int      `json:"groupCount"`
	GroupIndex    int      `json:"groupIndex"`
	GroupLabels   []string `json:"groupLabels,omitempty"`
	NeedGroupPick bool     `json:"needGroupPick"`
	// RowCount is len(folder rows)+len(file rows) in the compare table when a merge is active.
	CompareRowCount int `json:"compareRowCount"`
	// ApplyRunning is true while the on-disk executor is working (phase 5).
	ApplyRunning bool `json:"applyRunning"`
}

// ProgressEvent is emitted during ApplyExecute (phase 5).
type ProgressEvent struct {
	Phase   string `json:"phase"` // "running", "done", "cancelled", "error"
	Current int    `json:"current,omitempty"`
	Total   int    `json:"total,omitempty"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// FolderNode is a JSON tree node for the folder view.
type FolderNode struct {
	Path           string       `json:"path"`
	Name           string       `json:"name"`
	TotalSizeBytes int64        `json:"totalSizeBytes"`
	Children       []FolderNode `json:"children,omitempty"`
}

// MergePreview is a read-only view of a folder pair and file rows.
type MergePreview struct {
	LeftPath        string         `json:"leftPath"`
	RightPath       string         `json:"rightPath"`
	LeftLabel       string         `json:"leftLabel"`
	RightLabel      string         `json:"rightLabel"`
	LeftDupPct      float64        `json:"leftDupPct"`
	RightDupPct     float64        `json:"rightDupPct"`
	// LeftTotalFiles / RightTotalFiles count files directly in the compared folder only.
	LeftTotalFiles  int `json:"leftTotalFiles"`
	RightTotalFiles int `json:"rightTotalFiles"`
	// LeftTotalFilesRecursive / RightTotalFilesRecursive include all files under subfolders.
	LeftTotalFilesRecursive  int            `json:"leftTotalFilesRecursive"`
	RightTotalFilesRecursive int            `json:"rightTotalFilesRecursive"`
	// ChildFilesSize = sum of sizes of files directly in the folder (same scope as LeftTotalFiles).
	LeftChildFilesSizeBytes  int64 `json:"leftChildFilesSizeBytes"`
	RightChildFilesSizeBytes int64 `json:"rightChildFilesSizeBytes"`
	// TotalFilesSize = sum of sizes of all files under the folder tree (same scope as recursive file counts).
	LeftTotalFilesSizeBytes  int64 `json:"leftTotalFilesSizeBytes"`
	RightTotalFilesSizeBytes int64 `json:"rightTotalFilesSizeBytes"`
	Rows                     []MergeFileRow `json:"rows"`
	Note                     string         `json:"note,omitempty"`
	GroupCount               int            `json:"groupCount"`
	ShownGroupIdx            int            `json:"shownGroupIndex"`
	Interactive              bool           `json:"interactive"`
}

// MergeFileRow is one row in the compare table (files and subfolder rows).
type MergeFileRow struct {
	RowIndex int    `json:"rowIndex"`
	IsFolder bool   `json:"isFolder"`
	No       string `json:"no,omitempty"`
	// Action is none|deleteRight|deleteLeft|moveToRight|moveToLeft (matches core merge actions).
	Action    string `json:"action"`
	LeftName  string `json:"leftName,omitempty"`
	RightName string `json:"rightName,omitempty"`
	LeftPath  string `json:"leftPath,omitempty"`
	RightPath string `json:"rightPath,omitempty"`
	LeftMeta  string `json:"leftMeta,omitempty"`
	RightMeta string `json:"rightMeta,omitempty"`
	Kind      string `json:"kind"` // "folder", "pair", "left", "right"
	// SimilarityPct is 100 for exact hash match, -1 for similar pair (different hash), 0 for unique.
	SimilarityPct float64 `json:"similarityPct"`
	// LeftSizeBytes / RightSizeBytes are file sizes for merge pair rows (0 when file absent on that side).
	LeftSizeBytes  int64 `json:"leftSizeBytes"`
	RightSizeBytes int64 `json:"rightSizeBytes"`
}
