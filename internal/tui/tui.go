// Package tui provides an interactive terminal UI powered by Bubble Tea and Lip Gloss.
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	itemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	selectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(lipgloss.Color("#7D56F4")).
				Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			MarginTop(1)
)

var menuItems = []string{
	"Hello, World!",
	"About",
	"Quit",
}

// model holds the application state.
type model struct {
	cursor int
}

func initialModel() model {
	return model{}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(menuItems)-1 {
				m.cursor++
			}

		case "enter", " ":
			switch menuItems[m.cursor] {
			case "Quit":
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	s := titleStyle.Render("cli-prototype") + "\n\n"

	for i, item := range menuItems {
		if m.cursor == i {
			s += selectedItemStyle.Render("▸ "+item) + "\n"
		} else {
			s += itemStyle.Render("  "+item) + "\n"
		}
	}

	s += helpStyle.Render("↑/↓ navigate • enter select • q quit")
	return s
}

// Run starts the Bubble Tea program.
func Run() error {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}
	return nil
}
