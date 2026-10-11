package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/lasomethingsomething/cli-prototype/cmd"
	"github.com/lasomethingsomething/cli-prototype/internal/telemetry"
)

func main() {
	// First non-flag arg is the command name; "model-cli" if none.
	command := "model-cli"
	for _, a := range os.Args[1:] {
		if !strings.HasPrefix(a, "-") {
			command = a
			break
		}
	}

	tel := telemetry.Start("dev")
	ctx, span := tel.StartSpan(context.Background(), command)
	start := time.Now()

	err := cmd.Execute()

	took := time.Since(start).Seconds()
	exit := "ok"
	if err != nil {
		if errors.Is(err, cmd.ErrWizardCancelled) {
			exit = "cancelled"
		} else {
			exit = "error"
		}
	}
	tel.Record(ctx, command, exit, took)
	span.End()
	tel.Shutdown(context.Background())

	if err != nil {
		if errors.Is(err, cmd.ErrWizardCancelled) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
