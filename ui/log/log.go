package log

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type LogEntry struct {
	timestamp time.Time
	message   string
	isError   bool
}

type Model struct {
	entries    []LogEntry
	width      int
	height     int
	viewOffset int
	autoScroll bool
}
type LogMsg struct {
	Message string
	IsError bool
}

const (
	MaxEntries     = 100
	TimestampColor = lipgloss.Color("240")
	MessageColor   = lipgloss.Color("255")
)

func New() *Model {
	return &Model{
		entries:    make([]LogEntry, 0),
		autoScroll: true,
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case LogMsg:
		m.addLog(msg.Message, msg.IsError)
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			if m.viewOffset > 0 {
				m.viewOffset--
				m.autoScroll = false
			}
		case "down":
			maxOffset := max(0, len(m.entries)-m.height)
			if m.viewOffset < maxOffset {
				m.viewOffset++
			} else if m.viewOffset == maxOffset {
				m.autoScroll = true
			}
		}
	}
	return m, nil
}

func (m *Model) View() string {
	if len(m.entries) == 0 {
		return lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).
			Padding(0, 1).
			Render("No logs yet...")
	}

	// Calculate visible range
	start := m.viewOffset
	end := min(start+m.height, len(m.entries))

	visibleEntries := m.entries[start:end]

	var lines []string
	for _, entry := range visibleEntries {
		timestamp := entry.timestamp.Format("15:04:05")

		textColor := MessageColor
		if entry.isError {
			textColor = lipgloss.Color("160")
		}
		// Style the line
		styledLine := lipgloss.NewStyle().
			Foreground(TimestampColor).
			Render(fmt.Sprintf("[%s]", timestamp)) +
			" " +
			lipgloss.NewStyle().
				Foreground(textColor).
				Render(entry.message)

		lines = append(lines, styledLine)
	}

	content := ""
	for i, line := range lines {
		content += line
		if i < len(lines)-1 {
			content += "\n"
		}
	}

	lines = strings.Split(lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Render(content), "\n")

	if len(lines) > m.height {
		return strings.Join(lines[len(lines)-m.height:], "\n")
	}

	return strings.Join(lines, "\n")
}

func (m *Model) addLog(message string, isError bool) {
	entry := LogEntry{
		timestamp: time.Now(),
		message:   message,
		isError:   isError,
	}

	m.entries = append(m.entries, entry)

	// Maintain max entries limit
	if len(m.entries) > MaxEntries {
		m.entries = m.entries[len(m.entries)-MaxEntries:]
	}

	// Auto-scroll to bottom if enabled
	if m.autoScroll {
		maxOffset := max(0, len(m.entries)-m.height)
		m.viewOffset = maxOffset
	}
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height

	// Adjust view offset if needed
	maxOffset := max(0, len(m.entries)-height)
	if m.viewOffset > maxOffset {
		m.viewOffset = maxOffset
	}
}

func (m *Model) Info(message string) {
	m.addLog(message, false)
}

func (m *Model) Error(message string) {
	m.addLog(message, true)
}

type logger struct {
	p *tea.Program
}

func (l *logger) Info(message string) {
	if l.p != nil {
		l.p.Send(LogMsg{Message: message, IsError: false})
	}
}

func (l *logger) Error(message string) {
	if l.p != nil {
		l.p.Send(LogMsg{Message: message, IsError: true})
	}
}

func EventLogger(p *tea.Program) *logger {
	return &logger{p: p}
}
