package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
	"github.com/spf13/cobra"
)

// The ask* helpers implement the one rule every command follows for input:
// a value given on the command line wins, otherwise the user is prompted.
// "Given" means the flag was present (cmd.Flags().Changed), so an explicit
// empty value such as --cuda-min "" is respected and not prompted for again.
//
// When prompting is disabled (see detectInteractivity) the helpers keep the
// current value of dest: free-text inputs fall back to their defaults and
// commands check required ones with requireValues; selects fail immediately
// unless the current value is one of the options.

// askString sets *dest from the flag if it was given, otherwise prompts for
// it. The prompt is prefilled with the current value of *dest.
func askString(cmd *cobra.Command, flag string, dest *string, title, description string) error {
	if cmd.Flags().Changed(flag) {
		v, err := cmd.Flags().GetString(flag)
		if err != nil {
			return err
		}
		*dest = v
		return nil
	}
	if !interactive() {
		return nil
	}
	in := huh.NewInput().Title(title).Value(dest)
	if description != "" {
		in = in.Description(description)
	}
	return in.Run()
}

// askSelect sets *dest from the flag if it was given, otherwise prompts the
// user to pick one of options.
func askSelect(cmd *cobra.Command, flag string, dest *string, title, description string, options []string) error {
	return askSelectLabeled(cmd, flag, dest, title, description, huh.NewOptions(options...))
}

// askSelectLabeled is askSelect with explicit option labels, for choices
// whose stored value needs an explanation (e.g. MOF classes).
func askSelectLabeled(cmd *cobra.Command, flag string, dest *string, title, description string, options []huh.Option[string]) error {
	if cmd.Flags().Changed(flag) {
		v, err := cmd.Flags().GetString(flag)
		if err != nil {
			return err
		}
		*dest = v
		return nil
	}
	if !interactive() {
		var values []string
		for _, o := range options {
			if o.Value == *dest {
				return nil
			}
			values = append(values, fmt.Sprintf("%q", o.Value))
		}
		return fmt.Errorf("--%s is required (prompts are disabled: %s); choose one of %s", flag, nonInteractiveReason(), strings.Join(values, ", "))
	}
	sel := huh.NewSelect[string]().Title(title).Options(options...).Value(dest)
	if description != "" {
		sel = sel.Description(description)
	}
	return sel.Run()
}

// askSelectIfEmpty is askSelect for values that may already be known, e.g.
// from the saved config: the flag wins, then an existing non-empty *dest is
// kept, and only then is the user prompted.
func askSelectIfEmpty(cmd *cobra.Command, flag string, dest *string, title, description string, options []string) error {
	if !cmd.Flags().Changed(flag) && *dest != "" {
		return nil
	}
	return askSelect(cmd, flag, dest, title, description, options)
}

// toolOptions converts a tool category into prompt options whose labels carry
// the description and the "(recommended)" marker; the stored value is the name.
func toolOptions(opts workflow.ToolOptions) []huh.Option[string] {
	options := make([]huh.Option[string], 0, len(opts))
	for _, o := range opts {
		options = append(options, huh.NewOption(o.Label(), o.Name))
	}
	return options
}

// askSelectToolIfEmpty is askSelectIfEmpty for a tool category such as
// workflow.SignerOptions(): the choices and their labels come from one place.
func askSelectToolIfEmpty(cmd *cobra.Command, flag string, dest *string, title, description string, opts workflow.ToolOptions) error {
	if !cmd.Flags().Changed(flag) && *dest != "" {
		return nil
	}
	return askSelectLabeled(cmd, flag, dest, title, description, toolOptions(opts))
}

// askConfirm sets *dest from the bool flag if it was given, otherwise asks.
func askConfirm(cmd *cobra.Command, flag string, dest *bool, title, description string) error {
	if cmd.Flags().Changed(flag) {
		v, err := cmd.Flags().GetBool(flag)
		if err != nil {
			return err
		}
		*dest = v
		return nil
	}
	if !interactive() {
		return nil
	}
	c := huh.NewConfirm().Title(title).Value(dest)
	if description != "" {
		c = c.Description(description)
	}
	return c.Run()
}
