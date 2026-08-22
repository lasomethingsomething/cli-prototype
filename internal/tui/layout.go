package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Layout manages the overall TUI layout with header, content, and footer
type Layout struct {
	Title      string
	Content    string
	Footer     string
	Status     string
	StatusStyle lipgloss.Style
	Style      lipgloss.Style
}

// NewLayout creates a new layout
func NewLayout(title string) *Layout {
	return &Layout{
		Title: title,
		Content: "",
		Footer: "",
		Status: "",
		StatusStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF88")).
			Bold(true),
		Style: lipgloss.NewStyle().Padding(1, 2),
	}
}

// SetContent sets the main content
func (l *Layout) SetContent(content string) {
	l.Content = content
}

// SetFooter sets the footer
func (l *Layout) SetFooter(footer string) {
	l.Footer = footer
}

// SetStatus sets the status message
func (l *Layout) SetStatus(status string, isError bool) {
	l.Status = status
	if isError {
		l.StatusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5555")).
			Bold(true)
	} else {
		l.StatusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF88")).
			Bold(true)
	}
}

// Render renders the complete layout
func (l *Layout) Render() string {
	var sb strings.Builder
	
	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Bold(true).
		Padding(1, 2)
	
	if l.Title != "" {
		sb.WriteString(titleStyle.Render(l.Title))
		sb.WriteString("\n")
	}
	
	// Content
	if l.Content != "" {
		sb.WriteString(l.Content)
		sb.WriteString("\n")
	}
	
	// Status
	if l.Status != "" {
		sb.WriteString(l.StatusStyle.Render(l.Status))
		sb.WriteString("\n")
	}
	
	// Footer
	if l.Footer != "" {
		footerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			PaddingTop(1)
		sb.WriteString(footerStyle.Render(l.Footer))
	}
	
	return sb.String()
}

// ProgressBar represents a simple progress indicator
type ProgressBar struct {
	Current int
	Total   int
	Width   int
	Filled  string
	Empty   string
}

// NewProgressBar creates a new progress bar
func NewProgressBar(current, total, width int) *ProgressBar {
	return &ProgressBar{
		Current: current,
		Total:   total,
		Width:   width,
		Filled:  "█",
		Empty:   "░",
	}
}

// Render renders the progress bar
func (p *ProgressBar) Render() string {
	if p.Total == 0 {
		return ""
	}
	
	percentage := float64(p.Current) / float64(p.Total)
	filledWidth := int(percentage * float64(p.Width))
	
	filled := strings.Repeat(p.Filled, filledWidth)
	empty := strings.Repeat(p.Empty, p.Width-filledWidth)
	
	bar := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FF88")).
		Render(filled) + 
		lipgloss.NewStyle().
		Foreground(lipgloss.Color("#555555")).
		Render(empty)
	
	percentText := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Render(fmt.Sprintf(" %d%%", int(percentage*100)))
	
	return fmt.Sprintf("[%s]%s", bar, percentText)
}

// InfoBox creates a styled information box
type InfoBox struct {
	Title   string
	Message string
	Type    string // "info", "warning", "error", "success"
}

// NewInfoBox creates a new info box
func NewInfoBox(title, message, boxType string) *InfoBox {
	return &InfoBox{
		Title:   title,
		Message: message,
		Type:    boxType,
	}
}

// Render renders the info box
func (i *InfoBox) Render() string {
	var color string
	var icon string
	
	switch i.Type {
	case "success":
		color = "#00FF88"
		icon = "✓"
	case "warning":
		color = "#FFAA00"
		icon = "⚠"
	case "error":
		color = "#FF5555"
		icon = "✗"
	default: // "info"
		color = "#8888FF"
		icon = "ℹ"
	}
	
	boxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(color)).
		Padding(0, 1)
	
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(color)).
		Bold(true)
	
	return boxStyle.Render(headerStyle.Render(icon+" "+i.Title) + "\n" + i.Message)
}
