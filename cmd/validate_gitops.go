package cmd

import (
	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
	"github.com/spf13/cobra"
)

func newValidateGitOpsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "gitops",
		Short: "Validate artifact annotations for GitOps deployment",
		Long: `Pre-sync validation for GitOps tools (Argo CD, Flux): fetches the artifact manifest
and checks the Trust Profile annotations (signing, SBOM, provenance) and Infrastructure
Requirement annotations (runtime, accelerator, CUDA, memory). With --env, the air-gapped
and hybrid-cloud safety policies are applied as well.

Actual policy enforcement is delegated to external engines (Sigstore Policy Controller,
OPA/Gatekeeper, Kyverno); this command checks that they will find what they need.
Exit code is 0 on pass and 1 on fail, for use as a pre-sync hook.

Examples:
  model-cli validate gitops --artifact ghcr.io/my-org/my-model:latest
  model-cli validate gitops --artifact my-model:v1 --env air-gapped --quiet
  model-cli validate gitops --artifact my-model:v1 --env hybrid-cloud --region us-east-1 --json-output`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := outputOptionsFrom(cmd)
			env, _ := cmd.Flags().GetString("env")
			region, _ := cmd.Flags().GetString("region")

			artifact, annotations, err := fetchAnnotations(cmd, out)
			if err != nil {
				return err
			}
			// A GitOps pre-sync hook must block on data-residency mismatches.
			report := workflow.EvaluateArtifact("gitops", artifact, annotations, workflow.Policy{Environment: env, Region: region, Strict: true})
			return printReport(report, out, gitopsHints)
		},
	}
	addArtifactFlags(c)
	addOutputFlags(c)
	c.Flags().String("env", "", "Target environment: development, staging, production, air-gapped, hybrid-cloud")
	c.Flags().String("region", "", "Target region for hybrid-cloud validation (e.g., us-east-1, eu-west-1)")
	return c
}
