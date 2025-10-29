package progress

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	windowWidth  int
	windowHeight int
	dialogWidth  int
	progress     progress.Model
	current      int
	total        int
	message      string
	cancelled    bool
}

type ProgressUpdateMsg struct {
	Current int
	Total   int
	Message string
}

type ProgressCompleteMsg struct {
	Success bool
	Message string
}

type ProgressCancelMsg struct{}

const (
	ProgressBarWidth = 40
	ProgressColor    = lipgloss.Color("69")
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
	case ProgressUpdateMsg:
		m.current = msg.Current
		m.total = msg.Total
		m.message = msg.Message
		if m.total > 0 {
			m.progress.SetPercent(float64(m.current) / float64(m.total))
		}
		return m, nil
	case ProgressCompleteMsg:
		m.message = msg.Message
		if msg.Success {
			m.progress.SetPercent(1.0)
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.cancelled = true
			cmd = CancelProgress()
		}
	}

	// Update progress bar - handle FrameMsg for animation
	var progressCmd tea.Cmd
	progressModel, progressCmd := m.progress.Update(msg)
	if progressModel, ok := progressModel.(progress.Model); ok {
		m.progress = progressModel
	}
	if progressCmd != nil {
		cmd = tea.Batch(cmd, progressCmd)
	}
	return m, cmd
}

func CancelProgress() tea.Cmd {
	return func() tea.Msg {
		return ProgressCancelMsg{}
	}
}

func (m *Model) View() string {
	if m.windowWidth == 0 || m.total == 0 {
		return "Loading..."
	}

	progressBar := m.progress.ViewAs(float64(m.current) / float64(m.total))

	// Create content
	var content strings.Builder

	// Title
	title := "Executing File Tasks"
	if m.cancelled {
		title = "Cancelling..."
	}
	content.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Render(title))
	content.WriteString("\n\n")

	// Progress info
	if m.total > 0 {
		progressText := fmt.Sprintf("Progress: %d/%d tasks", m.current, m.total)
		content.WriteString(progressText)
		content.WriteString("\n")
	}

	// Progress bar
	content.WriteString(progressBar)
	content.WriteString("\n\n")

	// Current message
	if m.message != "" {
		// Truncate long messages
		message := m.message
		content.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render(message))
		content.WriteString("\n")
	}

	// Instructions
	if !m.cancelled {
		content.WriteString("\n")
		content.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("Press ESC to cancel"))
	}

	// Apply dialog styling
	dialogStyle := lipgloss.NewStyle().
		Width(m.dialogWidth).
		Height(m.windowHeight).
		Border(lipgloss.RoundedBorder(), true).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2)

	return dialogStyle.Render(content.String())
}

func New() *Model {
	p := progress.New(progress.WithDefaultGradient())
	p.Width = ProgressBarWidth
	p.ShowPercentage = true
	p.SetPercent(0.0) // Initialize at 0%

	return &Model{
		windowWidth:  50,
		windowHeight: 10,
		dialogWidth:  0, // Will be set by parent
		progress:     p,
		current:      0,
		total:        0,
		message:      "",
		cancelled:    false,
	}
}

func (m *Model) SetSize(width, height int) {
	m.windowWidth = width
	m.windowHeight = height
	m.progress.Width = width - 6
}

func (m *Model) SetDialogWidth(width int) {
	m.dialogWidth = width
}

func (m *Model) IsCancelled() bool {
	return m.cancelled
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
