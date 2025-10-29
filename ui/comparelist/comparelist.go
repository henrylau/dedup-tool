package comparelist

import (
	"fmt"
	"folder-similarity/core"
	"strconv"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	FolderAPathStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("229")).
				Background(lipgloss.Color("129"))
	FolderBPathStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("229")).
				Background(lipgloss.Color("64"))

	ActionIcons = []string{"", "⌦", "⌫", "⏵", "⏴"}
)

type Model struct {
	ready       bool
	folder1     *core.FolderSimilarity
	folder2     *core.FolderSimilarity
	table       table.Model
	help        help.Model
	width       int
	height      int
	keyMap      KeyMap
	filePairs   []core.MergeFilePair
	folderPairs []core.MergeFolderPair
}

type KeyMap struct {
	DeleteRight    key.Binding
	DeleteLeft     key.Binding
	MoveToRight    key.Binding
	MoveToLeft     key.Binding
	DeleteRightAll key.Binding
	DeleteLeftAll  key.Binding
	MoveToRightAll key.Binding
	MoveToLeftAll  key.Binding
	Clear          key.Binding
	ClearAll       key.Binding
	Apply          key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.DeleteRight,
		k.DeleteLeft,
		k.MoveToRight,
		k.MoveToLeft,
		k.Apply,
	}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.DeleteRight, k.DeleteLeft},
		{k.MoveToRight, k.MoveToLeft},
	}
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		DeleteRight: key.NewBinding(
			key.WithKeys("."),
			key.WithHelp(".", "delete right(⌦)"),
		),
		DeleteLeft: key.NewBinding(
			key.WithKeys(","),
			key.WithHelp(",", "delete left(⌫)"),
		),
		MoveToRight: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("⮕", "move to right"),
		),
		MoveToLeft: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("⮕", "move to left"),
		),
		DeleteRightAll: key.NewBinding(
			key.WithKeys(">"),
			key.WithHelp(">", "delete right all"),
		),
		DeleteLeftAll: key.NewBinding(
			key.WithKeys("<"),
			key.WithHelp("<", "delete left all"),
		),
		MoveToRightAll: key.NewBinding(
			key.WithKeys("shift+right"),
			key.WithHelp("shift+right", "move to right all"),
		),
		MoveToLeftAll: key.NewBinding(
			key.WithKeys("shift+left"),
			key.WithHelp("shift+left", "move to left all"),
		),
		Clear: key.NewBinding(
			key.WithKeys("c", "backspace"),
			key.WithHelp("c/⟵", "clear"),
		),
		ClearAll: key.NewBinding(
			key.WithKeys("C"),
			key.WithHelp("C", "clear all"),
		),
		Apply: key.NewBinding(
			key.WithKeys("A"),
			key.WithHelp("A", "apply"),
		),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m *Model) SetMergeFolderPair(mergeFolderPair *core.MergeFolderPair) {
	m.filePairs, m.folderPairs = nil, nil
	m.folder1, m.folder2 = nil, nil

	if mergeFolderPair == nil {
		m.updateItems()
		return
	}

	if mergeFolderPair.MatchType == core.MatchBothSide {
		if folder1, ok1 := mergeFolderPair.Folder1.(*core.FolderSimilarity); ok1 {
			m.folder1 = folder1
		}
		if folder2, ok2 := mergeFolderPair.Folder2.(*core.FolderSimilarity); ok2 {
			m.folder2 = folder2
		}
		m.filePairs = mergeFolderPair.FilePairs
		m.folderPairs = mergeFolderPair.FolderPairs
		m.table.SetCursor(0)
	} else {
		// TODO: handle only left or right
	}
	m.updateItems()

}

func (m *Model) updateItems() {
	if len(m.filePairs) == 0 && len(m.folderPairs) == 0 {
		m.table.SetRows([]table.Row{})
		return
	}
	rows := []table.Row{}
	// update folder pairs
	for _, pair := range m.folderPairs {
		rows = append(rows, table.Row{
			"",
			pair.GetName(0),
			pair.GetFileCount(0),
			pair.GetDuplicatedPercentage(0),
			ActionIcons[pair.Action],
			pair.GetName(1),
			pair.GetFileCount(1),
			pair.GetDuplicatedPercentage(1),
		})
	}
	// update file pairs
	for i, pair := range m.filePairs {
		rows = append(rows, table.Row{
			strconv.Itoa(i + 1),
			pair.GetName(0),
			pair.GetSize(0),
			pair.GetModified(0),
			ActionIcons[pair.Action],
			pair.GetName(1),
			pair.GetSize(1),
			pair.GetModified(1),
		})
	}

	m.table.SetRows(rows)
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.table.SetWidth(width)
	m.table.SetHeight(height)
	m.ready = true

	nameWidth := max(15, (width-68)/2)
	// update table column size
	columns := m.table.Columns()
	columns[1].Width = nameWidth
	columns[5].Width = nameWidth
	m.table.SetColumns(columns)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.table, _ = m.table.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// if no file pairs, do nothing
		if len(m.filePairs) == 0 {
			return &m, nil
		}
		switch {
		case key.Matches(msg, m.keyMap.DeleteRight):
			m.SetAction(m.table.Cursor(), core.ActionDeleteRight)
		case key.Matches(msg, m.keyMap.DeleteLeft):
			m.SetAction(m.table.Cursor(), core.ActionDeleteLeft)
		case key.Matches(msg, m.keyMap.MoveToRight):
			m.SetAction(m.table.Cursor(), core.ActionMoveToRight)
		case key.Matches(msg, m.keyMap.MoveToLeft):
			m.SetAction(m.table.Cursor(), core.ActionMoveToLeft)
		case key.Matches(msg, m.keyMap.Clear):
			m.SetAction(m.table.Cursor(), core.ActionNone)
		case key.Matches(msg, m.keyMap.ClearAll):
			m.ClearAllActions()
		case key.Matches(msg, m.keyMap.DeleteRightAll):
			m.SetAllActions(core.ActionDeleteRight)
		case key.Matches(msg, m.keyMap.DeleteLeftAll):
			m.SetAllActions(core.ActionDeleteLeft)
		case key.Matches(msg, m.keyMap.MoveToRightAll):
			m.SetAllActions(core.ActionMoveToRight)
		case key.Matches(msg, m.keyMap.MoveToLeftAll):
			m.SetAllActions(core.ActionMoveToLeft)
		case key.Matches(msg, m.keyMap.Apply):
			actions := m.GetActions()
			return &m, applyActions(actions)
		}

		m.updateItems()
	}
	return &m, nil
}

// Set Single Action of selected item
func (m *Model) SetAction(index int, action core.MergeAction) {
	if index < len(m.folderPairs) {
		m.folderPairs[m.table.Cursor()].SetAction(core.MergeAction(action))
	} else {
		m.filePairs[m.table.Cursor()-len(m.folderPairs)].SetAction(action)
	}
}

func (m *Model) SetAllActions(action core.MergeAction) {
	for i := range m.folderPairs {
		m.folderPairs[i].SetAction(core.MergeAction(action))
	}
	for i := range m.filePairs {
		m.filePairs[i].SetAction(action)
	}
}

func (m *Model) ClearAllActions() {
	for i := range m.folderPairs {
		m.folderPairs[i].SetAction(core.ActionNone)
	}
	for i := range m.filePairs {
		m.filePairs[i].SetAction(core.ActionNone)
	}
}

func (m Model) View() string {
	if !m.ready || m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	helpView := m.help.View(m.keyMap)

	pathInfo := ""

	if m.folder1 != nil && m.folder2 != nil {
		pathInfo = lipgloss.JoinHorizontal(
			lipgloss.Top,
			FolderAPathStyle.Width(m.width/2).Render(m.folder1.Path+fmt.Sprintf(" (cover %.02f%% - %d/%d)", m.folder1.DuplicatedPercentage(), m.folder1.DuplicateFileCount, m.folder1.FileCount)),
			FolderBPathStyle.Width(m.width/2).Render(m.folder2.Path+fmt.Sprintf(" (cover %.02f%% - %d/%d)", m.folder2.DuplicatedPercentage(), m.folder2.DuplicateFileCount, m.folder2.FileCount)),
		)
	}
	m.table.SetHeight(m.height - lipgloss.Height(pathInfo) - lipgloss.Height(helpView))

	return lipgloss.JoinVertical(lipgloss.Left, pathInfo, m.table.View(), helpView)
}

func (m *Model) GetActions() []core.FileActionTask {
	actions := []core.FileActionTask{}
	for _, pair := range m.filePairs {
		if pair.Action != core.ActionNone {
			actions = append(actions, pair.GetActionTask(m.folder1, m.folder2))
		}
	}

	for _, pair := range m.folderPairs {
		if pair.Action != core.ActionNone {
			actions = append(actions, pair.GetActionTask(m.folder1, m.folder2)...)
		}
	}

	actions = append(actions, core.FileActionTask{
		Action: core.DeleteEmptyFolder,
		Folder: m.folder1.Folder,
	})

	actions = append(actions, core.FileActionTask{
		Action: core.DeleteEmptyFolder,
		Folder: m.folder2.Folder,
	})
	return actions
}

func New() *Model {
	columns := []table.Column{
		{Title: "No.", Width: 3},
		{Title: "Name", Width: 15},
		{Title: "Size", Width: 8},
		{Title: "Modified", Width: 16},
		{Title: "A", Width: 1},
		{Title: "Name", Width: 15},
		{Title: "Size", Width: 8},
		{Title: "Modified", Width: 16},
	}

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)

	keyMap := DefaultKeyMap()

	m := &Model{
		keyMap: keyMap,
		table: table.New(
			table.WithColumns(columns),
			table.WithFocused(true),
			table.WithStyles(s),
		),
		help: help.New(),
	}

	return m
}

type ActionApplyMsg struct {
	Actions []core.FileActionTask
}

func applyActions(actions []core.FileActionTask) tea.Cmd {
	return func() tea.Msg {
		return ActionApplyMsg{
			Actions: actions,
		}
	}
}
