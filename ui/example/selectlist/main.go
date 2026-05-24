package main

import (
	"fmt"
	"log"

	"folder-similarity/ui/selectlistdialog"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	selectDialog *selectlistdialog.Model
	showDialog   bool
	result       string
	dialogMode   string // "single" or "multi"
	width        int
	height       int
}

var (
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true).
			MarginBottom(1)

	resultStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			MarginTop(1).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("57"))
)

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.showDialog {
			m.selectDialog.SetSize(msg.Width, msg.Height)
		}
		return m, nil

	case tea.KeyMsg:
		if !m.showDialog {
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "s":
				// Show single-select dialog
				options := []string{
					"Option 1",
					"Option 2",
					"Option 3",
					"Option 4",
					"Option 5",
				}
				m.selectDialog = selectlistdialog.New("Choose one option:", options, false)
				m.selectDialog.SetSize(m.width, m.height)
				m.showDialog = true
				m.dialogMode = "single"
				return m, nil
			case "m":
				// Show multi-select dialog
				options := []string{
					"Apple",
					"Banana",
					"Cherry",
					"Date",
					"Elderberry",
					"Fig",
					"Grape",
					"Kiwi",
					"Lemon",
					"Mango",
				}
				m.selectDialog = selectlistdialog.New("Choose multiple options:", options, true)
				m.selectDialog.SetSize(m.width, m.height)
				m.showDialog = true
				m.dialogMode = "multi"
				return m, nil
			}
		} else {
			// Handle dialog
			var updated tea.Model
			updated, cmd = m.selectDialog.Update(msg)
			m.selectDialog = updated.(*selectlistdialog.Model)
			return m, cmd
		}

	case selectlistdialog.CloseMsg:
		m.showDialog = false
		if msg.Confirmed {
			if len(msg.Selected) == 0 {
				m.result = "No options selected"
			} else if len(msg.Selected) == 1 {
				m.result = "Selected: " + msg.Selected[0]
			} else {
				m.result = fmt.Sprintf("Selected %d options: %v", len(msg.Selected), msg.Selected)
			}
		} else {
			m.result = "Dialog cancelled (Esc pressed)"
		}
		return m, nil
	}

	return m, nil
}

func (m model) View() string {
	if m.showDialog {
		return m.selectDialog.View()
	}

	content := titleStyle.Render("SelectList Dialog Example")

	content += "\n\n"
	content += "This example demonstrates the SelectListModel with both single-select and multi-select modes.\n\n"

	content += "Controls:\n"
	content += "• Press 's' for single-select dialog (radio buttons)\n"
	content += "• Press 'm' for multi-select dialog (checkboxes)\n"
	content += "• Press 'q' or Ctrl+C to quit\n\n"

	content += "Dialog Controls:\n"
	content += "• ↑/↓ - Navigate through options\n"
	content += "• Space - Toggle selection (multi-select) or select item (single-select)\n"
	content += "• Tab - Switch focus between list and OK button\n"
	content += "• Enter - Confirm selection\n"
	content += "• Esc - Cancel dialog\n\n"

	if m.result != "" {
		content += resultStyle.Render("Last Result:\n" + m.result)
	}

	return helpStyle.Render(content)
}

func main() {
	m := model{
		width:  80,
		height: 25,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
