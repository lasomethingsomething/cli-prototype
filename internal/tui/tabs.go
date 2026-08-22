package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Tab represents a single tab in the tab bar
type Tab struct {
	Name     string
	Content  string
	Active   bool
}

// TabBar represents a horizontal tab navigation bar
type TabBar struct {
	Tabs    []Tab
	Active  int
	Style   lipgloss.Style
	ActiveStyle lipgloss.Style
}

// NewTabBar creates a new tab bar with the given tab names
func NewTabBar(tabNames []string) *TabBar {
	tabs := make([]Tab, len(tabNames))
	for i, name := range tabNames {
		tabs[i] = Tab{
			Name:    name,
			Content: "",
			Active:  i == 0,
		}
	}
	
	return &TabBar{
		Tabs:    tabs,
		Active:  0,
		Style:   lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).Padding(0, 1),
		ActiveStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA")).Bold(true).Underline(true),
	}
}

// SetContent sets the content for a specific tab
func (t *TabBar) SetContent(index int, content string) {
	if index >= 0 && index < len(t.Tabs) {
		t.Tabs[index].Content = content
	}
}

// SetActive sets the active tab
func (t *TabBar) SetActive(index int) {
	if index >= 0 && index < len(t.Tabs) {
		t.Active = index
		for i := range t.Tabs {
			t.Tabs[i].Active = (i == index)
		}
	}
}

// Render renders the tab bar as a string
func (t *TabBar) Render() string {
	var sb strings.Builder
	
	// Render tab headers
	for i, tab := range t.Tabs {
		style := t.Style
		if tab.Active {
			style = t.ActiveStyle
		}
		sb.WriteString(style.Render(tab.Name))
		if i < len(t.Tabs)-1 {
			sb.WriteString("|")
		}
	}
	
	return sb.String()
}

// RenderWithContent renders the tab bar and the active tab's content
func (t *TabBar) RenderWithContent() string {
	var sb strings.Builder
	
	// Render tab headers
	tabHeaders := t.Render()
	sb.WriteString(tabHeaders)
	sb.WriteString("\n")
	
	// Render active tab content
	if t.Active >= 0 && t.Active < len(t.Tabs) {
		sb.WriteString(t.Tabs[t.Active].Content)
	}
	
	return sb.String()
}

// ContextPanel represents a panel that shows contextual information
type ContextPanel struct {
	Title   string
	Content string
	Style   lipgloss.Style
}

// NewContextPanel creates a new context panel
func NewContextPanel(title, content string) *ContextPanel {
	return &ContextPanel{
		Title:   title,
		Content: content,
		Style:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2),
	}
}

// Render renders the context panel
func (c *ContextPanel) Render() string {
	borderStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#55AAFF")).
		Bold(true)
	
	title := borderStyle.Render(c.Title)
	content := c.Style.Render(c.Content)
	
	return fmt.Sprintf("%s\n%s", title, content)
}
