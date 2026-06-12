package cmd

import (
	"fmt"

	"github.com/minhnc/easy-infra/internal/config"
	"github.com/minhnc/easy-infra/internal/profile"
	"github.com/minhnc/easy-infra/internal/project"
	"github.com/minhnc/easy-infra/internal/service"
	"github.com/minhnc/easy-infra/internal/state"
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
	if _, err := config.Load(paths.Config); err == nil {
		return fmt.Errorf("project already initialized (%s exists)", paths.Config)
	}

	cfg := config.Scaffold()
	if err := cfg.Save(paths.Config); err != nil {
		return err
	}

	prof, err := profile.Scaffold(reg, project.DefaultServices...)
	if err != nil {
		return err
	}
	defaultProfilePath := paths.ProfilePath("default")
	if err := prof.Save(defaultProfilePath); err != nil {
		return err
	}

	st := &state.State{ActiveProfile: "default"}
	if err := st.Save(paths.State); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(),
		"Initialized easy-infra project:\n  project config: %s\n  default profile: %s\n  state:          %s\nActive profile: default\n",
		paths.Config, defaultProfilePath, paths.State)
	return nil
}
