package cmd

import (
	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
	"github.com/spf13/cobra"
)

func newValidateAdmissionCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "admission",
		Short: "Evaluate an artifact for GitOps admission to an environment",
		Long: `Evaluate whether an artifact can be admitted to a target environment: fetches the
manifest from the registry and checks the Trust Profile, Infrastructure Requirements and
the environment's safety policies (air-gapped, hybrid-cloud with data residency).

By default a data-residency mismatch is a warning; --strict makes it block admission.
Signature and SBOM validity are not verified here - that is delegated to Sigstore Policy
Controller, cosign and friends.

Examples:
  model-cli validate admission --artifact ghcr.io/my-org/my-model:latest --env production
  model-cli validate admission --artifact my-model:v1 --env hybrid-cloud --region eu-west-1 --strict
  model-cli validate admission --artifact my-model:v1 --env air-gapped --json-output`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := outputOptionsFrom(cmd)
			strict, _ := cmd.Flags().GetBool("strict")

			var env string
			if err := askSelect(cmd, "env", &env, "Target environment:", "Destination environment for deployment",
				[]string{"development", "staging", "production", "air-gapped", "hybrid-cloud"}); err != nil {
				return err
			}
			var region string
			if cmd.Flags().Changed("region") || env == "hybrid-cloud" {
				if err := askString(cmd, "region", &region, "Target region:", "Target region for hybrid-cloud validation (e.g., us-east-1, eu-west-1)"); err != nil {
					return err
				}
			}

			artifact, annotations, err := fetchAnnotations(cmd, out)
			if err != nil {
				return err
			}
			report := workflow.EvaluateArtifact("admission", artifact, annotations, workflow.Policy{Environment: env, Region: region, Strict: strict})
			return printReport(report, out, gitopsHints)
		},
	}
	addArtifactFlags(c)
	addOutputFlags(c)
	c.Flags().String("env", "", "Target environment: development, staging, production, air-gapped, hybrid-cloud")
	c.Flags().String("region", "", "Target region for hybrid-cloud validation (e.g., us-east-1, eu-west-1)")
	c.Flags().Bool("strict", false, "Strict mode: block admission on data-residency mismatches")
	return c
}
