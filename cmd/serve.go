package cmd

import (
	"github.com/charmbracelet/huh"
	"github.com/lasomethingsomething/cli-prototype/config"
	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve a model using vLLM or KServe",
	Long: `Serve your ML model for inference using vLLM or KServe runtime.

Examples:
  model-cli serve
  model-cli serve --runtime vllm --model phi-4-mini --port 8080
  model-cli serve --runtime kserve --model my-model --host 0.0.0.0 --port 8080`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		// Interactive prompts
		var runtime string
		if cfg.Runtime == "" {
			if err := huh.NewSelect[string]().
				Title("Select runtime:").
				Options(huh.NewOptions("vllm", "kserve")...).
				Value(&runtime).
				Run(); err != nil {
				return err
			}
			cfg.Runtime = runtime
			config.Save(cfg)
		} else {
			runtime = cfg.Runtime
		}

		var modelPath string
		if err := huh.NewInput().
			Title("Model path:").
			Value(&modelPath).
			Run(); err != nil {
			return err
		}

		var host string
		if err := huh.NewInput().
			Title("Host:").
			Value(&host).
			Run(); err != nil {
			return err
		}

		var port string
		if err := huh.NewInput().
			Title("Port:").
			Value(&port).
			Run(); err != nil {
			return err
		}

		// Create serve workflow
		wf, err := workflow.NewServeWorkflow(runtime)
		if err != nil {
			return err
		}
		wf.SetServeInfo(modelPath, host, port)

		return wf.Run()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
