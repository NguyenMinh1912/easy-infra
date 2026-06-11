package cmd

import (
	"fmt"

	"github.com/minhnc/easy-infra/internal/project"
	"github.com/minhnc/easy-infra/internal/service"
	"github.com/spf13/cobra"
)

func newUseCmd(reg *service.Registry, paths project.Paths) *cobra.Command {
	return &cobra.Command{
		Use:   "use <profile>",
		Short: "Set the active profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return useProfile(cmd, reg, paths, args[0])
		},
	}
}

// useProfile sets name as the project's active profile and reports the change.
// It backs both `easy-infra use` and `easy-infra profile use`.
func useProfile(cmd *cobra.Command, reg *service.Registry, paths project.Paths, name string) error {
	proj, err := project.Load(paths, reg)
	if err != nil {
		return err
	}
	if err := proj.SetActiveProfile(name); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Active profile set to %q\n", name)
	return nil
}
