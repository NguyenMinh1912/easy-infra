package cmd

import (
	"fmt"

	"github.com/minhnc/easy-infra/internal/project"
	"github.com/minhnc/easy-infra/internal/service"
	"github.com/spf13/cobra"
)

func newBackupCmd(reg *service.Registry, paths project.Paths) *cobra.Command {
	return &cobra.Command{
		Use:   "backup",
		Short: "Back up data for the services in the active profile",
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
			fmt.Fprintf(out, "Backing up profile %q:\n", name)
			// As with apply, the actual backup mechanism is future work; this
			// loop is the per-service seam.
			for _, svcName := range sortedKeys(profile.Services) {
				fmt.Fprintf(out, "  - %s: would back up\n", svcName)
			}
			return nil
		},
	}
}
