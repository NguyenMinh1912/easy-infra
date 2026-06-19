package cmd

import (
	"fmt"

	"github.com/minhnc/easy-infra/internal/project"
	"github.com/minhnc/easy-infra/internal/service"
	"github.com/spf13/cobra"
)

func newInitCmd(reg *service.Registry, paths project.Paths) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize a project in the current folder",
		Long:  "Scaffold the project marker (easy-infra.yml), a default profile with its services, and the state file.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runInit(cmd, reg, paths)
		},
	}
}

func runInit(cmd *cobra.Command, reg *service.Registry, paths project.Paths) error {
	if err := project.Initialize(paths, reg); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(),
		"Initialized easy-infra project:\n  project config: %s\n  default profile: %s\n  state:          %s\nActive profile: default\n",
		paths.Config, paths.ProfilePath("default"), paths.State)
	return nil
}
