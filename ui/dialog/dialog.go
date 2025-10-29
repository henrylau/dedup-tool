package dialog

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	windowWidth  int
	windowHeight int
	message      string
	buttons      []string
	selected     int
}

type CloseMsg struct {
	Selected string
}

const (
	ButtonFocusColor = lipgloss.Color("229")
	ButtonColor      = lipgloss.Color("57")
)

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
		case "left":
			m.selected = max(m.selected-1, 0)
		case "right":
			m.selected = min(m.selected+1, len(m.buttons)-1)
		case "enter":
			cmd = CloseDialog(m.buttons[m.selected])
		}
	}

	return m, cmd
}

func CloseDialog(selected string) tea.Cmd {
	return func() tea.Msg {
		return CloseMsg{
			Selected: selected,
		}
	}
}

func (m *Model) View() string {
	foreStyle := lipgloss.NewStyle().
		Width(m.windowWidth).
		Border(lipgloss.RoundedBorder(), true).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2)

	buttonStyle := lipgloss.NewStyle().
		// Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Padding(0, 2).MarginRight(4)

	messages := m.message + "\n"
	buttons := ""

	for i, button := range m.buttons {
		color := ButtonColor
		if i == m.selected {
			color = ButtonFocusColor
		}
		buttons += buttonStyle.Background(color).Render(button)
	}

	layout := lipgloss.JoinVertical(lipgloss.Left, messages, buttons)

	return foreStyle.Render(layout)
}

func New(message string, buttons []string) *Model {
	return &Model{
		windowWidth:  50,
		windowHeight: 10,
		message:      message,
		buttons:      buttons,
	}
}

func (m *Model) SetMessage(message string) {
	m.message = message
}

func (m *Model) SetSize(width, height int) {
	m.windowWidth = width
	m.windowHeight = height
}
