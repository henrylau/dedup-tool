package main

import (
	"fmt"
	"folder-similarity/ui/tree"
	"log"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type MainModel struct {
	treeView *tree.Model
}

var (
	helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)
)

func (m MainModel) Init() tea.Cmd {
	return nil
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Update the tree view - no type assertion needed!
	m.treeView.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.treeView.Width = 30
		m.treeView.Height = msg.Height - 3
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "h":
			// Demonstrate programmatic highlight control
			// Find first item and highlight it
			if m.treeView.NodeList.Len() > 0 {
				firstElement := m.treeView.NodeList.Front()
				m.treeView.HighlightNode = firstElement
				m.treeView.CursorLine = 0
			}
			return m, nil
		case "l":
			// Demonstrate programmatic highlight control
			// Find last item and highlight it
			if m.treeView.NodeList.Len() > 0 {
				lastElement := m.treeView.NodeList.Back()
				m.treeView.HighlightNode = lastElement
				m.treeView.CursorLine = m.treeView.Height - 1
				if m.treeView.CursorLine < 0 {
					m.treeView.CursorLine = 0
				}
			}
			return m, nil
		default:
			return m, nil
		}
	default:
		return m, nil
	}
}

func (m MainModel) View() string {
	// return m.treeView.View()
	selected := ""
	if s := m.treeView.Selected(); s != nil {
		selected = s.GetName()
	}
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.treeView.View(),
		fmt.Sprintf("selected: %s, size: %dx%d", selected, m.treeView.Width, m.treeView.Height),
		helpStyle.Render("↑↓/kj: navigate • PgUp/Ctrl+U: half page up • PgDn/Ctrl+D: half page down • enter/space: select • h: highlight first • l: highlight last • q: quit"),
	)
}

type TItem struct {
	Name     string
	Children []*TItem
	parent   *TItem
}

var _ tree.Item = &TItem{}

func (i *TItem) GetName() string {
	return i.Name
}

// GetChildren implements tree.Item.
func (i *TItem) GetChildren() []tree.Item {
	var children []tree.Item
	for _, child := range i.Children {
		children = append(children, child)
	}
	return children
}

// Parent implements tree.Item.
func (i *TItem) Parent() tree.Item {
	return i.parent
}

func main() {
	// Demonstrate custom key bindings
	customKeys := tree.KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter", " "),
			key.WithHelp("↵/space", "select"),
		),
		HalfPageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+u", "u"),
			key.WithHelp("PgUp/Ctrl+U/u", "half page up"),
		),
		HalfPageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+d", "d"),
			key.WithHelp("PgDn/Ctrl+D/d", "half page down"),
		),
	}

	mainModel := MainModel{
		treeView: tree.WithKeyMap(customKeys), // Use custom key bindings
	}

	mainModel.treeView.AddItem(&TItem{Name: "root1", Children: []*TItem{
		{Name: "child1"},
		{Name: "child2", Children: []*TItem{
			{Name: "child21"},
			{Name: "child22文测试中文测试中文测试中文测试中文测试中文测试"},
			{Name: "dummy"},
			{Name: "dummy 12313 123 112231231231231231"},
			{Name: "dummy"},
			{Name: "dummy"},
			{Name: "dummy"},
			{Name: "dummy"},
			{Name: "dummy"},
			{Name: "dummy 12312312312312 3123213 123123 13123 "},
			{Name: "dummy"},
			{Name: "dummy中文测试中文测试中文测试中文测试中文测试中文测试"},
			{Name: "dummy"},
			{Name: "dummy"},
			{Name: "dummy"},
			{Name: "dummy"},
			{Name: "dummy"},
			{Name: "dummy"},
			{Name: "dummy"},
			{Name: "dummy"},
			{Name: "dummy"},
			{Name: "child23", Children: []*TItem{
				{Name: "child231"},
				{Name: "child232"},
				{Name: "child233"},
			}},
		}},
		{Name: "child3"},
		{Name: "child4", Children: []*TItem{
			{Name: "child41"},
			{Name: "child42"},
			{Name: "child43"},
		}},
	}})

	mainModel.treeView.AddItem(&TItem{Name: "root2", Children: []*TItem{
		{Name: "child4", Children: []*TItem{
			{Name: "child41"},
			{Name: "child42"},
			{Name: "child43"},
		}},
		{Name: "child5", Children: []*TItem{
			{Name: "child51", Children: []*TItem{
				{Name: "child511"},
				{Name: "child512"},
				{Name: "child513"},
			}},
			{Name: "child52"},
			{Name: "child53", Children: []*TItem{
				{Name: "child531"},
				{Name: "child532"},
				{Name: "child533"},
			}},
		}},
		{Name: "child6", Children: []*TItem{
			{Name: "child61"},
			{Name: "child62"},
			{Name: "child63"},
		}},
	}})

	p := tea.NewProgram(mainModel, tea.WithAltScreen())

	go func() {
		<-time.After(300 * time.Second)
		log.Fatal("10 seconds has elapsed")
	}()

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}

}
