package convert

import (
	"folder-similarity/app/dto"
	"folder-similarity/core"
	"strconv"
)

// BuildInteractiveMergePreview builds the compare table including folder rows, file rows, and action codes.
func BuildInteractiveMergePreview(
	merge *core.MergeFolderPair,
	note string,
	groupCount, shownGroupIdx int,
	interactive bool,
) dto.MergePreview {
	out := dto.MergePreview{
		Note:          note,
		GroupCount:    groupCount,
		ShownGroupIdx: shownGroupIdx,
		Interactive:   interactive,
		Rows:          nil,
	}
	if merge == nil {
		return out
	}
	if f1, ok := merge.Folder1.(*core.FolderSimilarity); ok {
		out.LeftPath = f1.Folder.Path
		out.LeftLabel = f1.Folder.Name
		out.LeftDupPct = safeDupPct(f1)
		if f1.Folder != nil {
			out.LeftTotalFiles = f1.Folder.GetDirectFileCount()
			out.LeftTotalFilesRecursive = f1.Folder.GetFileCount()
			out.LeftChildFilesSizeBytes = f1.Folder.GetDirectFilesSize()
			out.LeftTotalFilesSizeBytes = f1.Folder.GetTotalFilesSizeRecursive()
		}
	}
	if f2, ok := merge.Folder2.(*core.FolderSimilarity); ok {
		out.RightPath = f2.Folder.Path
		out.RightLabel = f2.Folder.Name
		out.RightDupPct = safeDupPct(f2)
		if f2.Folder != nil {
			out.RightTotalFiles = f2.Folder.GetDirectFileCount()
			out.RightTotalFilesRecursive = f2.Folder.GetFileCount()
			out.RightChildFilesSizeBytes = f2.Folder.GetDirectFilesSize()
			out.RightTotalFilesSizeBytes = f2.Folder.GetTotalFilesSizeRecursive()
		}
	}
	rowIdx := 0
	for _, fp := range merge.FolderPairs {
		row := dto.MergeFileRow{
			RowIndex:  rowIdx,
			IsFolder:  true,
			Action:    mergeActionString(fp.Action),
			Kind:      "folder",
			LeftName:  fp.GetName(0),
			RightName: fp.GetName(1),
			LeftMeta:  fp.GetFileCount(0) + "  " + fp.GetDuplicatedPercentage(0),
			RightMeta: fp.GetFileCount(1) + "  " + fp.GetDuplicatedPercentage(1),
		}
		switch fp.MatchType {
		case core.MatchBothSide:
			if f1, ok := fp.Folder1.(*core.FolderSimilarity); ok && f1.Folder != nil {
				row.LeftPath = f1.Folder.Path
			}
			if f2, ok := fp.Folder2.(*core.FolderSimilarity); ok && f2.Folder != nil {
				row.RightPath = f2.Folder.Path
			}
		case core.MatchOnlyLeft:
			if f1, ok := fp.Folder1.(*core.Folder); ok {
				row.LeftPath = f1.Path
			}
		case core.MatchOnlyRight:
			if f2, ok := fp.Folder2.(*core.Folder); ok {
				row.RightPath = f2.Path
			}
		}
		out.Rows = append(out.Rows, row)
		rowIdx++
	}
	for i, fp := range merge.FilePairs {
		row := dto.MergeFileRow{
			RowIndex: rowIdx,
			IsFolder: false,
			No:       strconv.Itoa(i + 1),
			Action:   mergeActionString(fp.Action),
			LeftMeta:  fp.GetSize(0),
			RightMeta: fp.GetSize(1),
		}
		if fp.File1 != nil {
			row.LeftSizeBytes = fp.File1.Size
		}
		if fp.File2 != nil {
			row.RightSizeBytes = fp.File2.Size
		}
		if fp.File1 != nil && fp.File2 != nil {
			row.Kind = "pair"
			row.LeftName = fp.File1.Name
			row.LeftPath = fp.File1.Path
			row.RightName = fp.File2.Name
			row.RightPath = fp.File2.Path
			if fp.File1.Hash == fp.File2.Hash {
				row.SimilarityPct = 100
			} else {
				row.SimilarityPct = -1
			}
		} else if fp.File1 != nil {
			row.Kind = "left"
			row.LeftName = fp.File1.Name
			row.LeftPath = fp.File1.Path
		} else if fp.File2 != nil {
			row.Kind = "right"
			row.RightName = fp.File2.Name
			row.RightPath = fp.File2.Path
		}
		out.Rows = append(out.Rows, row)
		rowIdx++
	}
	return out
}

func mergeActionString(a core.MergeAction) string {
	switch a {
	case core.ActionNone:
		return "none"
	case core.ActionDeleteRight:
		return "deleteRight"
	case core.ActionDeleteLeft:
		return "deleteLeft"
	case core.ActionMoveToRight:
		return "moveToRight"
	case core.ActionMoveToLeft:
		return "moveToLeft"
	default:
		return "none"
	}
}

func safeDupPct(f *core.FolderSimilarity) float64 {
	if f == nil || f.FileCount == 0 {
		return 0
	}
	return f.DuplicatedPercentage()
}
