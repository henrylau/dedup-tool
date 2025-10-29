package core

import (
	"fmt"
	"strconv"
)

type MergeAction int

const (
	ActionNone MergeAction = iota
	ActionDeleteRight
	ActionDeleteLeft
	ActionMoveToRight
	ActionMoveToLeft
)

type MatchType int

const (
	MatchBothSide MatchType = iota
	MatchOnlyLeft
	MatchOnlyRight
)

type MergeFolderPair struct {
	Folder1   interface{}
	Folder2   interface{}
	Action    MergeAction
	MatchType MatchType

	FilePairs   []MergeFilePair
	FolderPairs []MergeFolderPair
}

type MergeFilePair struct {
	File1  *File
	File2  *File
	Action MergeAction
}

func (m *MergeFolderPair) GetName(index int) string {
	if m.MatchType == MatchBothSide {
		f1, ok1 := m.Folder1.(*FolderSimilarity)
		f2, ok2 := m.Folder2.(*FolderSimilarity)
		if index == 0 && ok1 {
			return "📁" + f1.Folder.Name
		} else if index == 1 && ok2 {
			return "📁" + f2.Folder.Name
		}
	} else if m.MatchType == MatchOnlyLeft && index == 0 {
		if f1, ok1 := m.Folder1.(*Folder); ok1 {
			return "📁" + f1.Name
		}
	} else if m.MatchType == MatchOnlyRight && index == 1 {
		if f2, ok2 := m.Folder2.(*Folder); ok2 {
			return "📁" + f2.Name
		}
	}

	return ""
}

func (m *MergeFolderPair) GetFileCount(index int) string {
	if m.MatchType == MatchBothSide {
		f1, ok1 := m.Folder1.(*FolderSimilarity)
		f2, ok2 := m.Folder2.(*FolderSimilarity)
		if index == 0 && ok1 {
			return strconv.Itoa(f1.Folder.GetFileCount()) + " files"
		} else if index == 1 && ok2 {
			return strconv.Itoa(f2.Folder.GetFileCount()) + " files"
		}
	} else if m.MatchType == MatchOnlyLeft && index == 0 {
		if f1, ok1 := m.Folder1.(*Folder); ok1 {
			return strconv.Itoa(f1.GetFileCount()) + " files"
		}
	} else if m.MatchType == MatchOnlyRight && index == 1 {
		if f2, ok2 := m.Folder2.(*Folder); ok2 {
			return strconv.Itoa(f2.GetFileCount()) + " files"
		}
	}
	return ""
}

func (m *MergeFolderPair) GetDuplicatedPercentage(index int) string {
	if m.MatchType == MatchBothSide {
		f1, ok1 := m.Folder1.(*FolderSimilarity)
		f2, ok2 := m.Folder2.(*FolderSimilarity)
		if index == 0 && ok1 {
			return fmt.Sprintf("%.02f%%", f1.DuplicatedPercentage())
		} else if index == 1 && ok2 {
			return fmt.Sprintf("%.02f%%", f2.DuplicatedPercentage())
		}
	}
	return ""
}

func (m *MergeFolderPair) SetAction(action MergeAction) {
	m.Action = action
	switch m.MatchType {
	case MatchOnlyLeft:
		if action == ActionMoveToLeft || action == ActionDeleteRight {
			m.Action = ActionNone
		}
	case MatchOnlyRight:
		if action == ActionMoveToRight || action == ActionDeleteLeft {
			m.Action = ActionNone
		}
	}
}

func (m *MergeFolderPair) GetActionTask(folder1, folder2 *FolderSimilarity) []FileActionTask {
	actions := []FileActionTask{}
	if m.Action == ActionNone {
		return actions
	}
	if m.MatchType == MatchBothSide {
		folder1, ok1 := m.Folder1.(*FolderSimilarity)
		folder2, ok2 := m.Folder2.(*FolderSimilarity)
		if !ok1 || !ok2 {
			return actions
		}
		matchedPairs, f1only, f2only := GetMatchedFilePairs(folder1, folder2)

		for _, pair := range m.FolderPairs {
			pair.SetAction(m.Action)
			actions = append(actions, pair.GetActionTask(folder1, folder2)...)
		}

		switch m.Action {
		case ActionDeleteRight:
			// delete duplicated files in folder2
			for _, pair := range matchedPairs {
				actions = append(actions, FileActionTask{
					Action: Delete,
					File:   pair[1],
				})
			}
			for _, file := range f2only {
				actions = append(actions, FileActionTask{
					Action:       Delete,
					File:         file,
					NotDuplicate: true,
				})
			}
			actions = append(actions, FileActionTask{
				Action: DeleteEmptyFolder,
				Folder: folder2.Folder,
			})
		case ActionDeleteLeft:
			// delete duplicated files in folder1
			for _, pair := range matchedPairs {
				actions = append(actions, FileActionTask{
					Action: Delete,
					File:   pair[0],
				})
			}
			for _, file := range f1only {
				actions = append(actions, FileActionTask{
					Action:       Delete,
					File:         file,
					NotDuplicate: true,
				})
			}
			actions = append(actions, FileActionTask{
				Action: DeleteEmptyFolder,
				Folder: folder1.Folder,
			})
		case ActionMoveToRight:
			// delete duplicated files in folder1
			for _, pair := range matchedPairs {
				actions = append(actions, FileActionTask{
					Action: Delete,
					File:   pair[0],
				})
			}
			for _, file := range f1only {
				actions = append(actions, FileActionTask{
					Action:       Move,
					File:         file,
					TargetFolder: folder2.Folder,
				})
			}
			actions = append(actions, FileActionTask{
				Action: DeleteEmptyFolder,
				Folder: folder1.Folder,
			})
		case ActionMoveToLeft:
			// delete duplicated files in folder2
			for _, pair := range matchedPairs {
				actions = append(actions, FileActionTask{
					Action: Delete,
					File:   pair[1],
				})
			}
			for _, file := range f2only {
				actions = append(actions, FileActionTask{
					Action:       Move,
					File:         file,
					TargetFolder: folder1.Folder,
				})
			}
			actions = append(actions, FileActionTask{
				Action: DeleteEmptyFolder,
				Folder: folder2.Folder,
			})
		}

	} else if m.MatchType == MatchOnlyLeft {
		folder1, ok1 := m.Folder1.(*Folder)
		if !ok1 {
			return actions
		}
		switch m.Action {
		case ActionDeleteLeft:
			return []FileActionTask{
				{Action: DeleteFolder, Folder: folder1, NotDuplicate: true},
			}
		case ActionMoveToRight:
			return []FileActionTask{
				{Action: MoveFolder, Folder: folder1, TargetFolder: folder2.Folder},
			}
		}
	} else if m.MatchType == MatchOnlyRight {
		folder2, ok2 := m.Folder2.(*Folder)
		if !ok2 {
			return actions
		}
		switch m.Action {
		case ActionDeleteRight:
			return []FileActionTask{
				{Action: DeleteFolder, Folder: folder2, NotDuplicate: true},
			}
		case ActionMoveToLeft:
			return []FileActionTask{
				{Action: MoveFolder, Folder: folder2, TargetFolder: folder1.Folder},
			}
		}
	}
	return actions
}

func (m *MergeFilePair) GetName(index int) string {
	if index == 0 && m.File1 != nil {
		return m.File1.Name
	} else if index == 1 && m.File2 != nil {
		return m.File2.Name
	}
	return ""
}

func (m *MergeFilePair) GetSize(index int) string {
	if index == 0 && m.File1 != nil {
		return FormatFileSize(m.File1.Size)
	} else if index == 1 && m.File2 != nil {
		return FormatFileSize(m.File2.Size)
	}
	return ""
}

func (m *MergeFilePair) GetModified(index int) string {
	if index == 0 && m.File1 != nil {
		return m.File1.ModTime.Format("2006-01-02 15:04")
	} else if index == 1 && m.File2 != nil {
		return m.File2.ModTime.Format("2006-01-02 15:04")
	}
	return ""
}

func (m *MergeFilePair) SetAction(action MergeAction) {
	if (action == ActionMoveToLeft || action == ActionDeleteRight) && m.File2 == nil {
		m.Action = ActionNone
		return
	}
	if (action == ActionMoveToRight || action == ActionDeleteLeft) && m.File1 == nil {
		m.Action = ActionNone
		return
	}
	if action == ActionMoveToLeft && m.File1 != nil {
		m.Action = ActionDeleteRight
		return
	}
	if action == ActionMoveToRight && m.File2 != nil {
		m.Action = ActionDeleteLeft
		return
	}
	m.Action = action
}

func (m *MergeFilePair) GetActionTask(folder1, folder2 *FolderSimilarity) FileActionTask {
	switch m.Action {
	case ActionDeleteRight:
		return FileActionTask{
			Action: Delete,
			File:   m.File2,
		}
	case ActionDeleteLeft:
		return FileActionTask{
			Action: Delete,
			File:   m.File1,
		}
	case ActionMoveToRight:
		var name string
		if m.File2 != nil {
			name = m.File2.Name
		}
		return FileActionTask{
			Action:       Move,
			File:         m.File1,
			TargetFolder: folder2.Folder,
			TargetName:   name,
		}
	case ActionMoveToLeft:
		var name string
		if m.File1 != nil {
			name = m.File1.Name
		}
		return FileActionTask{
			Action:       Move,
			File:         m.File2,
			TargetFolder: folder1.Folder,
			TargetName:   name,
		}
	}
	return FileActionTask{}
}
