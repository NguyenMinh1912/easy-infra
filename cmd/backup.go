package cmd

import (
	"errors"
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
			name, prof, err := proj.ActiveProfile()
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Backing up profile %q:\n", name)
			// Each service backs itself up through Service.Backup. As with apply,
			// the actual backup mechanism is future work, so providers report
			// ErrNotImplemented for now; we surface that as the intended action.
			ctx := cmd.Context()
			for _, svcName := range sortedKeys(prof.Services) {
				svc, ok := reg.Get(svcName)
				if !ok {
					return fmt.Errorf("unknown service %q", svcName)
				}
				spec := service.Spec{
					Profile:    name,
					Definition: proj.Config.Services[svcName],
					Env:        prof.Services[svcName],
				}
				switch err := svc.Backup(ctx, spec); {
				case errors.Is(err, service.ErrNotImplemented):
					fmt.Fprintf(out, "  - %s: would back up from %s\n", svcName, endpoint(spec.Env))
				case err != nil:
					return fmt.Errorf("backing up %s: %w", svcName, err)
				default:
					fmt.Fprintf(out, "  - %s: backed up from %s\n", svcName, endpoint(spec.Env))
				}
			}
			return nil
		},
	}
}
