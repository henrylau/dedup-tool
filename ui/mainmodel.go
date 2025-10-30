package ui

import (
	"context"
	"fmt"
	"folder-similarity/core"
	"folder-similarity/ui/comparelist"
	"folder-similarity/ui/dialog"
	logui "folder-similarity/ui/log"
	"folder-similarity/ui/progress"
	"folder-similarity/ui/selectlistdialog"
	"folder-similarity/ui/tree"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	overlay "github.com/rmhubbert/bubbletea-overlay"
)

type FocusState int

const (
	TreeFocus FocusState = iota
	ListFocus
	LogFocus
	SelectListDialogFocus = 98
	DialogFocus           = 99
	ProgressFocus         = 100
)

type MainModel struct {
	treeView     tree.Model
	fileListView *comparelist.Model
	logView      *logui.Model
	focus        FocusState
	width        int
	height       int
	ready        bool
	rootPath     string

	storage           core.Storage
	similarityChecker *core.SimilarityChecker
	rootFolder        *FolderItemWrapper
	selectedFolder    *FolderItemWrapper

	actionConfirmDialog *dialog.Model
	progressDialog      *progress.Model
	selectListDialog    *selectlistdialog.Model
	overlay             tea.Model
	pendingActions      []core.FileActionTask
	logger              core.Logger
	executorCancel      context.CancelFunc
	currentExecutor     *core.Executor
	mergeFolderPair     core.MergeFolderPair

	// Temporary storage for similarity groups when showing selection dialog
	pendingSimilarityGroups [][2]*core.FolderSimilarity
}

type ProgressCompleteMsg struct {
	Success bool
	Message string
}

type ProgressCancelMsg struct{}

// Progress command that listens to executor progress channel
func listenProgress(progressChan <-chan core.ProgressUpdate) tea.Cmd {
	return func() tea.Msg {
		update := <-progressChan
		return progress.ProgressUpdateMsg{
			Current: update.Current,
			Total:   update.Total,
			Message: update.Message,
		}
	}
}

var (
	borderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("69")).
			Padding(0, 0)
	focusedBorderStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("205")).
				Padding(0, 0)
)

func (m *MainModel) Init() tea.Cmd {
	return nil
}

func (m *MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Calculate dimensions for the new layout
		treeWidth := m.width/4 - 2
		rightWidth := m.width/4*3 - 2
		treeHeight := m.height - 2

		// Account for borders: each bordered component needs 2 extra lines (top + bottom border)
		// Split right side: FileListView gets 2/3, LogView gets 1/3
		// Subtract 4 lines total for borders (2 for file list + 2 for log view)
		availableHeight := m.height - 4
		logHeight := min(10, availableHeight/3)
		fileListHeight := availableHeight - logHeight

		m.treeView.Width = treeWidth
		m.treeView.Height = treeHeight
		m.fileListView.SetSize(rightWidth, fileListHeight)
		m.progressDialog.SetSize(rightWidth*3/4, 8)
		m.actionConfirmDialog.SetSize(rightWidth*3/4, 8)
		m.selectListDialog.SetSize(rightWidth*3/4, min(15, m.height-4))
		m.logView.SetSize(rightWidth, logHeight)
		m.ready = true
		return m, nil
	case dialog.CloseMsg:
		if msg.Selected == "OK" {
			if m.logger == nil {
				m.logView.Error("Logger is not set")
				return m, nil
			}
			// Switch to progress dialog
			m.focus = ProgressFocus
			// Configure progress dialog width to 75% of table view width
			rightWidth := m.width/4*3 - 2
			dialogWidth := int(float64(rightWidth) * 0.75)
			m.progressDialog.SetDialogWidth(dialogWidth)
			m.overlay = overlay.New(m.progressDialog, m.fileListView, overlay.Center, overlay.Center, 0, 0)

			// Create cancellable context
			ctx, cancel := context.WithCancel(context.Background())
			m.executorCancel = cancel

			executor := core.NewExecutor(m.storage, m.rootPath, m.pendingActions, m.logger)
			m.currentExecutor = executor

			go func() {
				err := executor.Execute(ctx)
				if err != nil {
					if err == context.Canceled {
						m.logView.Info("Task execution cancelled")
					} else {
						m.logView.Error(err.Error())
					}
				} else {
					m.logView.Info(fmt.Sprintf("Execution completed with %d tasks", len(m.pendingActions)))
				}
			}()

			// Start listening to progress updates
			return m, listenProgress(executor.ProgressChannel())
		} else {
			m.pendingActions = nil
			m.focus = TreeFocus
		}
		return m, nil
	case selectlistdialog.CloseMsg:
		if msg.Confirmed && len(msg.Selected) > 0 {
			// Find the selected group index
			selectedIndex := -1
			for i, option := range m.selectListDialog.GetOptions() {
				if option == msg.Selected[0] {
					selectedIndex = i
					break
				}
			}

			if selectedIndex >= 0 && selectedIndex < len(m.pendingSimilarityGroups) {
				m.mergeFolderPair = m.similarityChecker.GenerateMergeFolderPair(m.pendingSimilarityGroups[selectedIndex][0], m.pendingSimilarityGroups[selectedIndex][1])
				m.fileListView.SetMergeFolderPair(&m.mergeFolderPair)

				// Check if we should switch to ListFocus
				if m.selectedFolder != nil && len(m.selectedFolder.GetChildren()) == 0 {
					m.focus = ListFocus
				} else {
					m.focus = TreeFocus
				}
			}
		} else {
			// User cancelled - return to tree focus
			m.focus = TreeFocus
		}
		// Clear pending groups
		m.pendingSimilarityGroups = nil
		return m, nil
	case logui.LogMsg:
		l, cmd := m.logView.Update(msg)
		if logModel, ok := l.(*logui.Model); ok {
			m.logView = logModel
		}
		return m, cmd
	case progress.ProgressUpdateMsg:
		p, cmd := m.progressDialog.Update(msg)
		if progressModel, ok := p.(*progress.Model); ok {
			m.progressDialog = progressModel
		}
		if msg.Current == msg.Total {
			return m, tea.Batch(cmd, func() tea.Msg {
				return progress.ProgressCompleteMsg{
					Success: true,
					Message: "Execution completed",
				}
			})
		}
		// Continue listening for more progress updates
		return m, tea.Batch(cmd, listenProgress(m.currentExecutor.ProgressChannel()))
	case progress.ProgressCompleteMsg:
		p, cmd := m.progressDialog.Update(msg)
		if progressModel, ok := p.(*progress.Model); ok {
			m.progressDialog = progressModel
		}
		// Auto-close progress dialog after completion
		m.focus = TreeFocus

		// refresh the tree data
		m.Refresh()
		return m, cmd
	case progress.ProgressCancelMsg:
		// Cancel the executor
		if m.executorCancel != nil {
			m.executorCancel()
		}
		m.focus = TreeFocus
		m.overlay = overlay.New(m.actionConfirmDialog, m.fileListView, overlay.Center, overlay.Center, 0, 0)
		return m, nil
	case comparelist.ActionApplyMsg: // Handle apply actions
		m.HandleApplyActions(msg)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			if m.focus < 3 {
				m.focus = (m.focus + 1) % 3 // Changed from % 2 to % 3 to include LogFocus
			}
			return m, nil
		}

		if m.focus == TreeFocus {
			v, _ := m.treeView.Update(msg)
			m.treeView = *v

			switch msg.String() {
			// Filter tree view
			case "f":
				highlightedItem := m.treeView.HighLightedItem()
				if m.treeView.HasFilter() {
					m.treeView.SetFilter(nil)
				} else {
					m.treeView.SetFilter(m.TreeFilter())
				}

				m.treeView.SetItems([]tree.Item{m.rootFolder})
				if highlightedItem != nil {
					m.treeView.MoveToItem(highlightedItem)
				}

				// Select folder
			case "enter":
				m.HandleTreeFolderSelected(m.treeView.Selected())
			}
		} else if m.focus == ListFocus {
			l, cmd := m.fileListView.Update(msg)
			if msg.String() == "o" {
				if m.mergeFolderPair.Folder1 != nil {
					folder1, ok1 := m.mergeFolderPair.Folder1.(*core.FolderSimilarity)
					if ok1 {
						m.OpenFileExplorer(folder1.Folder.Path)
					}
				}
				if m.mergeFolderPair.Folder2 != nil {
					folder2, ok2 := m.mergeFolderPair.Folder2.(*core.FolderSimilarity)
					if ok2 {
						m.OpenFileExplorer(folder2.Folder.Path)
					}
				}
			} else if msg.String() == "s" {
				switch m.storage.(type) {
				case *core.MemoryStorage:
					memoryStorage := m.storage.(*core.MemoryStorage)
					jsonData, err := memoryStorage.ExportStorage()
					if err != nil {
						m.logView.Error(err.Error())
					}
					m.logView.Info("Save the file list to db.json")
					err = os.WriteFile("db.json", jsonData, 0644)
					if err != nil {
						m.logView.Error(err.Error())
					}
				}
			}
			if fileListView, ok := l.(*comparelist.Model); ok {
				m.fileListView = fileListView
			}
			if cmd != nil {
				return m, cmd
			}
		} else if m.focus == DialogFocus {
			d, cmd := m.actionConfirmDialog.Update(msg)
			if dialogModel, ok := d.(*dialog.Model); ok {
				m.actionConfirmDialog = dialogModel
			}
			return m, cmd
		} else if m.focus == ProgressFocus {
			p, cmd := m.progressDialog.Update(msg)
			if progressModel, ok := p.(*progress.Model); ok {
				m.progressDialog = progressModel
			}
			return m, cmd
		} else if m.focus == SelectListDialogFocus {
			s, cmd := m.selectListDialog.Update(msg)
			if selectListModel, ok := s.(*selectlistdialog.Model); ok {
				m.selectListDialog = selectListModel
			}
			return m, cmd
		} else if m.focus == LogFocus {
			l, cmd := m.logView.Update(msg)
			if logModel, ok := l.(*logui.Model); ok {
				m.logView = logModel
			}
			return m, cmd
		}
	}
	return m, nil
}

func (m *MainModel) TreeFilter() func(item tree.Item) bool {
	return func(item tree.Item) bool {
		folder, ok := item.(*FolderItemWrapper)
		if !ok {
			return false
		}
		return m.similarityChecker.ContainsSimilarityGroup(folder.Path)
	}
}

func (m *MainModel) View() string {
	if !m.ready {
		return "Loading..."
	}

	treeViewStyle, tableViewStyle, logViewStyle := borderStyle, borderStyle, borderStyle

	if m.focus == TreeFocus {
		treeViewStyle = focusedBorderStyle
	} else if m.focus == ListFocus {
		tableViewStyle = focusedBorderStyle
	} else if m.focus == LogFocus {
		logViewStyle = focusedBorderStyle
	}

	mainContent := ""
	if m.focus == DialogFocus || m.focus == ProgressFocus || m.focus == SelectListDialogFocus {
		mainContent = tableViewStyle.Render(m.overlay.View())
	} else {
		mainContent = tableViewStyle.Render(m.fileListView.View())
	}

	// Create right side layout: FileListView on top, LogView on bottom
	rightSide := lipgloss.JoinVertical(
		lipgloss.Left,
		mainContent,
		logViewStyle.Render(m.logView.View()),
	)

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		treeViewStyle.Render(m.treeView.View()),
		rightSide,
	)
}

// NewMainModel creates a new MainModel instance
func NewMainModel() *MainModel {
	m := &MainModel{
		treeView:     tree.New(),
		fileListView: comparelist.New(),
		logView:      logui.New(),
		focus:        TreeFocus,
	}

	m.actionConfirmDialog = dialog.New("", []string{"OK", "Cancel"})
	m.progressDialog = progress.New()
	m.selectListDialog = selectlistdialog.New("Select folder pair to compare:", []string{}, false)

	m.treeView.SetFilter(m.TreeFilter())
	m.overlay = overlay.New(m.actionConfirmDialog, m.fileListView, overlay.Center, overlay.Center, 0, 0)
	return m
}

// SetStorage sets the storage for the model
func (m *MainModel) SetStorage(storage core.Storage) {
	m.storage = storage
}

// SetSimilarityChecker sets the similarity checker for the model
func (m *MainModel) SetSimilarityChecker(checker *core.SimilarityChecker) {
	m.similarityChecker = checker
}

// SetRootPath sets the root path for the model
func (m *MainModel) SetRootPath(path string) {
	m.rootPath = path
}

// SetLogger sets the logger for the model
func (m *MainModel) SetLogger(logger core.Logger) {
	m.logger = logger
}

// SetRootFolder sets the root folder for the model
func (m *MainModel) SetRootFolder(folder *FolderItemWrapper) {
	m.rootFolder = folder
	m.treeView.AddItem(m.rootFolder)
}

// GetRootPath returns the root path
func (m *MainModel) GetRootPath() string {
	return m.rootPath
}

// GetStorage returns the storage
func (m *MainModel) GetStorage() core.Storage {
	return m.storage
}

func (m *MainModel) HandleTreeFolderSelected(selectedItem tree.Item) {
	if selectedItem != nil {
		if folder, ok := selectedItem.(*FolderItemWrapper); ok {
			childCount := len(folder.GetChildren())

			m.selectedFolder = folder
			groups := m.similarityChecker.GetSimilarityFolderGroup(folder.Path)

			if len(groups) > 1 {
				// Multiple groups found - show selection dialog
				m.pendingSimilarityGroups = groups
				options := make([]string, len(groups))
				for i, group := range groups {
					targetPath := group[1].Path
					f1Duplicated := group[0].DuplicateFileCount
					f1Total := group[0].FileCount
					f1Coverage := group[0].DuplicatedPercentage()
					f2Duplicated := group[1].DuplicateFileCount
					f2Total := group[1].FileCount
					f2Coverage := group[1].DuplicatedPercentage()
					options[i] = fmt.Sprintf("%s (F1: %d/%d %.1f%% | F2: %d/%d %.1f%%)", targetPath, f1Duplicated, f1Total, f1Coverage, f2Duplicated, f2Total, f2Coverage)
				}
				m.selectListDialog.SetMessage(fmt.Sprintf("Target folder: %s, Select folder pair to compare: ", folder.Path))
				m.selectListDialog.SetOptions(options)
				m.focus = SelectListDialogFocus
				m.overlay = overlay.New(m.selectListDialog, m.fileListView, overlay.Center, overlay.Center, 0, 0)
			} else if len(groups) == 1 {
				m.mergeFolderPair = m.similarityChecker.GenerateMergeFolderPair(groups[0][0], groups[0][1])
				m.fileListView.SetMergeFolderPair(&m.mergeFolderPair)
				if childCount == 0 {
					m.focus = ListFocus
				}
			} else {
				m.fileListView.SetMergeFolderPair(nil)
			}
		}
	} else {
		m.fileListView.SetMergeFolderPair(nil)
	}
}

func (m *MainModel) HandleApplyActions(msg comparelist.ActionApplyMsg) {
	moveCount, deleteCount, replaceCount, nonDuplicateDeleteCount, deleteFolderCount, moveFolderCount := 0, 0, 0, 0, 0, 0
	for _, action := range msg.Actions {
		switch action.Action {
		case core.Move:
			if action.TargetName == "" {
				moveCount++
			} else {
				replaceCount++
			}
		case core.Delete:
			deleteCount++
			if action.NotDuplicate {
				nonDuplicateDeleteCount++
			}
		case core.DeleteFolder:
			deleteFolderCount++
		case core.MoveFolder:
			moveFolderCount++
		}
	}

	message := fmt.Sprintf("Apply following actions:\nMove %d files, delete %d files, replace %d files\nDelete  %d  Non-duplicate files, delete %d folders, move %d folders", moveCount, deleteCount, replaceCount, nonDuplicateDeleteCount, deleteFolderCount, moveFolderCount)
	m.actionConfirmDialog.SetMessage(message)
	m.overlay = overlay.New(m.actionConfirmDialog, m.fileListView, overlay.Center, overlay.Center, 0, 0)
	m.focus = DialogFocus
	m.pendingActions = msg.Actions
}

func (m *MainModel) OpenFileExplorer(path string) {
	switch runtime.GOOS {
	case "windows":
		exec.Command("explorer", path).Start()
	case "darwin":
		exec.Command("open", filepath.Join(m.rootPath, path)).Start()
	case "linux":
		exec.Command("xdg-open", path).Start()
	default:
		// unsupported OS, do nothing
	}
}

func (m *MainModel) Refresh() {
	currentNode := m.treeView.Selected()

	m.similarityChecker.CalculateSimilarity(m.storage)
	m.treeView.SetItems([]tree.Item{m.rootFolder})

	if currentNode != nil {
		if node, ok := currentNode.(tree.Item); ok {
			m.treeView.MoveToItem(node)
		}
	}
	m.fileListView.SetMergeFolderPair(nil)
}
