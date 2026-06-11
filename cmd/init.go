package cmd

import (
	"fmt"

	"github.com/minhnc/easy-infra/internal/config"
	"github.com/minhnc/easy-infra/internal/project"
	"github.com/minhnc/easy-infra/internal/service"
	"github.com/minhnc/easy-infra/internal/state"
	"github.com/spf13/cobra"
)

// scaffoldServices are the services included in the scaffolded default profile.
var scaffoldServices = []string{"postgres", "redis"}

func newInitCmd(reg *service.Registry, paths project.Paths) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize a project in the current folder",
		Long:  "Scaffold the YAML config and create the JSON state file for a new project.",
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

	cfg, err := config.Scaffold(reg, "default", scaffoldServices...)
	if err != nil {
		return err
	}
	if err := cfg.Save(paths.Config); err != nil {
		return err
	}

	st := &state.State{ActiveProfile: "default"}
	if err := st.Save(paths.State); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Initialized easy-infra project:\n  config: %s\n  state:  %s\nActive profile: default\n", paths.Config, paths.State)
	return nil
}
