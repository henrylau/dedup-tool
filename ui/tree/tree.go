// Package tree provides a tree view component for displaying hierarchical data
// with keyboard navigation and expand/collapse functionality.
package tree

import (
	"container/list"
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Item represents a node in the tree structure.
type Item interface {
	GetName() string
	GetChildren() []Item
	Parent() Item
}

// KeyMap defines the keyboard bindings for tree navigation.
type KeyMap struct {
	Up           key.Binding
	Down         key.Binding
	Left         key.Binding
	Right        key.Binding
	Enter        key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
}

// DefaultKeyMap returns the default keyboard bindings for tree navigation.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "move down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("←", "jump to parent"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("↵", "select"),
		),
		HalfPageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+u"),
			key.WithHelp("PgUp/Ctrl+U", "half page up"),
		),
		HalfPageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+d"),
			key.WithHelp("PgDn/Ctrl+D", "half page down"),
		),
		Right: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("→", "expand"),
		),
	}
}

// Model represents the tree view state and behavior.
type Model struct {
	Width  int
	Height int

	KeyMap KeyMap

	HighlightNode *list.Element
	NodeList      *list.List
	SelectedNode  *treeNode
	CursorLine    int // Line position of highlighted node within viewport (0 to Height-1)
	filter        func(item Item) bool
	hasFilter     bool
}

const (
	// Default colors for selected items
	selectedBgColor = "69"
	selectedFgColor = "230"
	// Tree symbols
	expandedIcon  = "▾ "
	collapsedIcon = "▸ "
	leafIcon      = "  "
	// Text truncation
	ellipsis = "..."
)

var (
	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color(selectedBgColor)).
			Foreground(lipgloss.Color(selectedFgColor))
	stickyStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("240")).
			Foreground(lipgloss.Color("255")).
			Bold(true)
)

// New creates a new tree model with default settings.
func New() Model {
	return Model{
		KeyMap:   DefaultKeyMap(),
		NodeList: list.New(),
	}
}

// WithKeyMap creates a new tree model with custom key bindings.
func WithKeyMap(keyMap KeyMap) *Model {
	return &Model{
		KeyMap:   keyMap,
		NodeList: list.New(),
	}
}

// View renders the tree component.
func (t Model) View() string {
	treeView := t.renderListView()

	return lipgloss.NewStyle().Height(t.Height).Render(treeView)
}

// Init initializes the tree model.
func (t Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the tree model.
func (t *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, t.KeyMap.Up):
			t.MoveUp(1)
		case key.Matches(msg, t.KeyMap.Down):
			t.MoveDown(1)
		case key.Matches(msg, t.KeyMap.Left):
			t.JumpToParent()
		case key.Matches(msg, t.KeyMap.HalfPageUp):
			t.MoveUp(t.Height / 2)
		case key.Matches(msg, t.KeyMap.HalfPageDown):
			t.MoveDown(t.Height / 2)
		case key.Matches(msg, t.KeyMap.Right):
			t.ExpandOrCollapse(t.HighlightNode)
		case key.Matches(msg, t.KeyMap.Enter):
			t.SelectedEnter()
		}
	}
	return t, nil
}

// MoveUp moves the highlight up by the specified number of steps.
func (t *Model) MoveUp(step int) {
	if step <= 0 {
		return
	}
	for i := 0; i < step && t.HighlightNode != nil && t.HighlightNode.Prev() != nil; i++ {
		t.HighlightNode = t.HighlightNode.Prev()
		// Only decrement cursor line if not at top of viewport
		if t.CursorLine > 0 {
			t.CursorLine--
		}
	}
}

// MoveDown moves the highlight down by the specified number of steps.
func (t *Model) MoveDown(step int) {
	if step <= 0 {
		return
	}
	for i := 0; i < step && t.HighlightNode != nil && t.HighlightNode.Next() != nil; i++ {
		t.HighlightNode = t.HighlightNode.Next()
		// Only increment cursor line if not at bottom of viewport
		if t.CursorLine < t.Height-1 {
			t.CursorLine++
		}
	}
}

// JumpToParent moves the highlight to the parent node.
func (t *Model) JumpToParent() {
	if t.HighlightNode == nil {
		return
	}

	node, ok := t.HighlightNode.Value.(*treeNode)
	if !ok || node.Parent == nil {
		return
	}

	// Find the parent by searching backwards from current position
	// Parent nodes are always before their children in the list
	for e := t.HighlightNode.Prev(); e != nil; e = e.Prev() {
		if e.Value.(*treeNode) == node.Parent {
			t.HighlightNode = e
			t.CursorLine = t.Height / 2 // Start at middle of viewport
			return
		}
	}
}

// SelectedEnter handles the enter key press on the highlighted node.
func (t *Model) SelectedEnter() {
	if t.HighlightNode != nil {
		node, ok := t.HighlightNode.Value.(*treeNode)
		if !ok {
			return
		}
		if node.HasChild {
			t.ExpandOrCollapse(t.HighlightNode)
			t.SelectedNode = node
		} else {
			t.SelectedNode = node
		}
	}
}

// AddItem adds a new item to the tree.
func (t *Model) AddItem(item Item) {
	newNode := &treeNode{
		Name:     item.GetName(),
		Layer:    0,
		HasChild: item.GetChildren() != nil,
		Expanded: false,
		Item:     item,
	}
	t.NodeList.PushBack(newNode)

	// Set initial highlight if this is the first item
	if t.NodeList.Len() == 1 {
		t.HighlightNode = t.NodeList.Back()
		t.CursorLine = 0 // Start at top of viewport
	}
}

func (t *Model) SetItems(items []Item) {
	t.NodeList = list.New()
	for _, item := range items {
		t.AddItem(item)
	}
}

// Selected returns the currently selected item.
func (t *Model) Selected() Item {
	if t.SelectedNode == nil {
		return nil
	}
	return t.SelectedNode.Item
}

func (t *Model) HighLightedItem() Item {
	if t.HighlightNode == nil {
		return nil
	}
	if i, ok := t.HighlightNode.Value.(*treeNode); ok && i.Item != nil {
		return i.Item
	}
	return nil
}

func (t *Model) SetFilter(filter func(item Item) bool) {
	if filter == nil {
		t.hasFilter = false
		t.filter = nil
	} else {
		t.hasFilter = true
		t.filter = filter
	}
}

func (t Model) HasFilter() bool {
	return t.hasFilter
}

// MoveToItem programmatically navigates to and highlights the specified item.
// It expands all parent nodes in the path and collapses nodes not on the path.
func (t *Model) MoveToItem(item Item) error {
	if item == nil {
		t.HighlightNode = nil
		t.CursorLine = 0
		return nil
	}

	// Build path from item to root using Item.Parent()
	path := []Item{}
	for p := item; p != nil; p = p.Parent() {
		path = append(path, p)
	}
	// Reverse to get root-to-item order
	slices.Reverse(path)

	// Collapse all nodes not on the path
	for e := t.NodeList.Front(); e != nil; e = e.Next() {
		node, ok := e.Value.(*treeNode)
		if !ok {
			continue
		}
		if node.Expanded {
			// Check if this node is on the path
			onPath := false
			for _, pathItem := range path {
				if node.Item == pathItem {
					onPath = true
					break
				}
			}
			if !onPath {
				t.ExpandOrCollapse(e) // Collapse it
			}
		}
	}

	// Expand all nodes on the path and set highlight as we go
	// This ensures parent is highlighted before expanding, preventing child not found issues
	for i, pathItem := range path {
		for e := t.NodeList.Front(); e != nil; e = e.Next() {
			node, ok := e.Value.(*treeNode)
			if !ok {
				continue
			}
			if node.Item == pathItem {
				// Set highlight to current node in path
				t.HighlightNode = e

				// Expand if it has children and is not the final target
				if i < len(path)-1 && node.HasChild && !node.Expanded {
					t.ExpandOrCollapse(e)
				}
				break
			}
		}
	}

	// Set final cursor position
	if t.HighlightNode != nil {
		t.CursorLine = t.Height / 2
		if t.CursorLine >= t.Height {
			t.CursorLine = t.Height - 1
		}
		return nil
	}

	return fmt.Errorf("item not found in tree")
}

// ExpandOrCollapse toggles the expansion state of a node.
func (t *Model) ExpandOrCollapse(listItem *list.Element) {
	if listItem == nil {
		return
	}
	node, ok := listItem.Value.(*treeNode)
	if !ok {
		return
	}

	if node.HasChild && !node.Expanded {
		node.Expanded = true
		currentItem := listItem

		for _, child := range node.Item.GetChildren() {
			newNode := &treeNode{
				Name:     child.GetName(),
				Layer:    node.Layer + 1,
				HasChild: child.GetChildren() != nil,
				Expanded: false,
				Item:     child,
				Parent:   node,
			}

			if t.hasFilter && !t.filter(child) {
				continue
			}

			currentItem = t.NodeList.InsertAfter(newNode, currentItem)
		}
	} else if node.HasChild && node.Expanded {
		node.Expanded = false

		currentItem := listItem.Next()
		for currentItem != nil {
			childNode, ok := currentItem.Value.(*treeNode)
			if !ok || childNode.Layer <= node.Layer {
				break
			}

			newNext := currentItem.Next()
			t.NodeList.Remove(currentItem)
			currentItem = newNext
		}
	}
}

// treeNode represents a single node in the tree.
type treeNode struct {
	Name     string
	Layer    int
	HasChild bool
	Expanded bool
	Item     Item
	Parent   *treeNode
}

func (t *Model) renderListView() string {
	if t.HighlightNode == nil {
		return ""
	}

	var lines []string
	currentNode, ok := t.HighlightNode.Value.(*treeNode)
	if !ok {
		return ""
	}

	// For sticky parent nodes calculation
	minLayer := currentNode.Layer
	withEllipsis := false
	parentNodes := map[*treeNode]int{}

	// Calculate how many nodes to render before the highlighted node
	// CursorLine is the position of highlight within viewport (0 to Height-1)
	// We need to render CursorLine nodes before the highlight
	nodesBefore := t.CursorLine

	// Collect nodes before highlight
	for i, n := 0, t.HighlightNode.Prev(); i < nodesBefore && n != nil; i++ {
		node, ok := n.Value.(*treeNode)
		if !ok {
			break
		}

		lines = append(lines, node.render(t.Width))

		// For sticky parent nodes calculation
		if node.Layer < minLayer {
			minLayer = node.Layer
		}
		if i == 0 && node.Layer == currentNode.Layer {
			withEllipsis = true
		}
		if node.HasChild {
			parentNodes[node] = nodesBefore - i
		}

		n = n.Prev()
	}
	// Reverse to get correct order
	slices.Reverse(lines)

	// Add highlighted node
	lines = append(lines, selectedStyle.Width(t.Width).Render(currentNode.render(t.Width)))

	// Render remaining nodes to fill viewport
	remainingLines := t.Height - len(lines)
	for i, n := 0, t.HighlightNode.Next(); i < remainingLines && n != nil; i++ {
		node, ok := n.Value.(*treeNode)
		if !ok {
			break
		}

		lines = append(lines, node.render(t.Width))
		n = n.Next()
	}

	// Render sticky parent nodes
	if minLayer > 0 {
		// Replace the top lines with sticky parent nodes
		parents := t.getParents(currentNode)
		slices.Reverse(parents)
		stickyHeader := []string{}
		for _, parent := range parents {
			parentNodeIndex, ok := parentNodes[parent]
			if !ok || parentNodeIndex <= len(stickyHeader)+1 {
				stickyHeader = append(stickyHeader, stickyStyle.Render(parent.render(t.Width)))
			}
		}
		if withEllipsis {
			stickyHeader = append(stickyHeader, strings.Repeat(" ", currentNode.Layer+2)+ellipsis)
		}
		// lines = append(stickyHeader, lines...)
		if t.CursorLine > len(stickyHeader) {
			lines = append(stickyHeader, lines[len(stickyHeader):]...)
		} else {

			lines = append(stickyHeader, lines[t.CursorLine:len(lines)-len(stickyHeader)+t.CursorLine]...)
		}
	}

	return strings.Join(lines, "\n")
}

func (t *Model) getParents(node *treeNode) []*treeNode {
	var parents []*treeNode

	for n := node.Parent; n != nil; n = n.Parent {
		parents = append(parents, n)
	}

	return parents
}

func (n *treeNode) render(maxWidth int) string {
	indent := strings.Repeat(" ", n.Layer)

	var icon string
	if n.HasChild {
		if n.Expanded {
			icon = expandedIcon
		} else {
			icon = collapsedIcon
		}
	} else {
		icon = leafIcon
	}

	// prefixLen := len(indent) + len(icon)
	name := n.Name
	// if len(name) > maxWidth-prefixLen && maxWidth > 0 {
	// 	name = name[:maxWidth-prefixLen-len(ellipsis)] + ellipsis
	// }

	return lipgloss.NewStyle().MaxWidth(maxWidth).Render(fmt.Sprintf("%s%s%s", indent, icon, name))
}
