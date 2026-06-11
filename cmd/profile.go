package cmd

import (
	"fmt"

	"github.com/minhnc/easy-infra/internal/project"
	"github.com/minhnc/easy-infra/internal/service"
	"github.com/spf13/cobra"
)

func newProfileCmd(reg *service.Registry, paths project.Paths) *cobra.Command {
	profile := &cobra.Command{
		Use:   "profile",
		Short: "Manage profiles",
		Long:  "List and inspect the environment profiles defined for the project.",
	}
	profile.AddCommand(newProfileListCmd(reg, paths))
	return profile
}

func newProfileListCmd(reg *service.Registry, paths project.Paths) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List the project's profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			proj, err := project.Load(paths, reg)
			if err != nil {
				return err
			}
			names, err := proj.Profiles()
			if err != nil {
				return err
			}
			if len(names) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no profiles found")
				return nil
			}
			out := cmd.OutOrStdout()
			active := proj.State.ActiveProfile
			for _, name := range names {
				marker := "  "
				if name == active {
					marker = "* "
				}
				fmt.Fprintf(out, "%s%s\n", marker, name)
			}
			return nil
		},
	}
}
