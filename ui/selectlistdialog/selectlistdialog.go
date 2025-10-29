package selectlistdialog

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	windowWidth  int
	windowHeight int
	selected     int
	list         *list.Model
	multiSelect  bool
	listFocus    bool
	options      []string
}

type CloseMsg struct {
	Selected  []string
	Confirmed bool
}

const (
	ButtonFocusColor = lipgloss.Color("229")
	ButtonColor      = lipgloss.Color("57")
)

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(0)
	itemStyle         = lipgloss.NewStyle()
	selectedItemStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
)

type item struct {
	option   string
	selected bool
}

func (i item) FilterValue() string { return "" }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	style := itemStyle
	if index == m.Index() {
		style = selectedItemStyle
	}

	selected := " "
	if i.selected {
		selected = "*"
	}
	text := fmt.Sprintf("[%s] %s", selected, i.option)

	fmt.Fprint(w, style.Render(text))
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			cmd = CloseDialog(nil)
		case "tab":
			m.listFocus = !m.listFocus
		case " ":
			m.SelectItem(m.list.Index())
		case "enter":
			cmd = CloseDialog(m.GetSelected())
		default:
			if m.listFocus {
				// TODO: list update
				updated, cmd := m.list.Update(msg)
				m.list = &updated
				return m, cmd
			}
		}
	}

	return m, cmd
}

func (m *Model) GetSelected() []string {
	var selected []string

	for _, listItem := range m.list.Items() {
		if item, ok := listItem.(item); ok && item.selected {
			selected = append(selected, item.option)
		}
	}

	return selected
}

func CloseDialog(selected []string) tea.Cmd {
	confirmed := len(selected) > 0
	return func() tea.Msg {
		return CloseMsg{
			Selected:  selected,
			Confirmed: confirmed,
		}
	}
}

func (m *Model) SelectItem(index int) {
	if !m.multiSelect {
		// deselect all items
		for i, listItem := range m.list.Items() {
			if item, ok := listItem.(item); ok && item.selected {
				item.selected = false
				m.list.SetItem(i, item)
			}
		}
	}

	if item, ok := m.list.Items()[index].(item); ok {
		item.selected = !item.selected
		m.list.SetItem(index, item)
	}
}

func (m *Model) View() string {
	foreStyle := lipgloss.NewStyle().
		Width(m.windowWidth-2).
		Height(m.windowHeight-2).
		Border(lipgloss.RoundedBorder(), true).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2)

	ButtonStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("57")).
		Padding(0, 2).MarginRight(4)

	if m.listFocus {
		ButtonStyle = ButtonStyle.Background(lipgloss.Color("229"))
	}

	listView := m.list.View()

	layout := lipgloss.JoinVertical(lipgloss.Left, listView, ButtonStyle.Render("OK"))

	return foreStyle.Render(layout)
}

func New(message string, options []string, multiSelect bool) *Model {
	l := list.New(nil, itemDelegate{}, 0, 0)

	items := []list.Item{}
	for i, option := range options {
		selected := false
		if i == 0 && !multiSelect {
			selected = true
		}
		items = append(items, item{option: option, selected: selected})
	}
	l.SetItems(items)
	l.Title = message
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle
	l.SetWidth(50)
	l.SetHeight(20)
	// l.SetSize(50, 20)

	return &Model{
		windowWidth:  50,
		windowHeight: 10,
		options:      options,
		multiSelect:  multiSelect,
		list:         &l,
		listFocus:    true,
	}
}

func (m *Model) SetMessage(message string) {
	m.list.Title = message
}

func (m *Model) SetSize(width, height int) {
	m.windowWidth = width
	m.windowHeight = height
	m.list.SetSize(width-8, height-6)
}

func (m *Model) SetOptions(options []string) {
	m.options = options
	items := []list.Item{}
	for i, option := range options {
		selected := false
		if i == 0 && !m.multiSelect {
			selected = true
		}
		items = append(items, item{option: option, selected: selected})
	}
	m.list.SetItems(items)
}

func (m *Model) GetOptions() []string {
	return m.options
}
