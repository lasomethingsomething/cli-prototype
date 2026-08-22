// Package tui provides utilities for running interactive TUI programs
package tui

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// RunContextModel runs an interactive context model as a modal
// The user can explore tabs and then continue
func RunContextModel(model *ContextModel) error {
	// Create a tea program with the context model
	p := tea.NewProgram(model)

	// Run the program
	_, err := p.Run()
	return err
}

// RunModel runs any tea.Model as a full-screen program
func RunModel(model tea.Model) error {
	p := tea.NewProgram(model)
	_, err := p.Run()
	return err
}

// RunWithOptions runs a tea.Model with custom options
func RunWithOptions(model tea.Model, options ...tea.ProgramOption) error {
	allOptions := []tea.ProgramOption{
		tea.WithOutput(os.Stdout),
		tea.WithInput(os.Stdin),
	}
	allOptions = append(allOptions, options...)

	p := tea.NewProgram(model, allOptions...)
	_, err := p.Run()
	return err
}
