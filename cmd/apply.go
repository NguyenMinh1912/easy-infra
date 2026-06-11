package cmd

import (
	"fmt"

	"github.com/minhnc/easy-infra/internal/project"
	"github.com/minhnc/easy-infra/internal/service"
	"github.com/spf13/cobra"
)

func newApplyCmd(reg *service.Registry, paths project.Paths) *cobra.Command {
	return &cobra.Command{
		Use:   "apply",
		Short: "Reconcile the active profile (provision/start its services)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			proj, err := project.Load(paths, reg)
			if err != nil {
				return err
			}
			name, profile, err := proj.ActiveProfile()
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Applying profile %q:\n", name)
			// Provisioning via Docker is intentionally not wired up yet; the
			// loop establishes the per-service seam where each provider will
			// reconcile its own service.
			for _, svcName := range sortedKeys(profile.Services) {
				fmt.Fprintf(out, "  - %s: would provision\n", svcName)
			}
			return nil
		},
	}
}
