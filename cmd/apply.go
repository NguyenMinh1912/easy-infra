package cmd

import (
	"errors"
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
			name, prof, err := proj.ActiveProfile()
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Applying profile %q:\n", name)
			// Each service reconciles itself through Service.Apply. Docker-backed
			// provisioning is future work, so providers report ErrNotImplemented
			// for now; we surface that as the intended action rather than failing.
			ctx := cmd.Context()
			for _, svcID := range sortedKeys(prof.Services) {
				entry := prof.Services[svcID]
				svcType := entry.ResolveType(svcID)
				svc, ok := reg.Get(svcType)
				if !ok {
					return fmt.Errorf("unknown service %q", svcType)
				}
				spec := service.Spec{
					Profile:    name,
					Definition: entry.Config,
					Env:        entry.Config,
				}
				switch err := svc.Apply(ctx, spec); {
				case errors.Is(err, service.ErrNotImplemented):
					fmt.Fprintf(out, "  - %s: would provision at %s\n", svcID, endpoint(spec.Env))
				case err != nil:
					return fmt.Errorf("applying %s: %w", svcID, err)
				default:
					fmt.Fprintf(out, "  - %s: applied at %s\n", svcID, endpoint(spec.Env))
				}
			}
			return nil
		},
	}
}
