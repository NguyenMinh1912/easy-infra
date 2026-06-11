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
		Long:  "List, add, remove, and switch the environment profiles defined for the project.",
	}
	profile.AddCommand(
		newProfileListCmd(reg, paths),
		newProfileAddCmd(reg, paths),
		newProfileRemoveCmd(reg, paths),
		newProfileUseCmd(reg, paths),
	)
	return profile
}

func newProfileAddCmd(reg *service.Registry, paths project.Paths) *cobra.Command {
	return &cobra.Command{
		Use:   "add <profile>",
		Short: "Add a profile with default service configuration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			proj, err := project.Load(paths, reg)
			if err != nil {
				return err
			}
			name := args[0]
			if _, err := proj.AddProfile(name); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Added profile %q (%s)\n", name, paths.ProfilePath(name))
			return nil
		},
	}
}

func newProfileRemoveCmd(reg *service.Registry, paths project.Paths) *cobra.Command {
	return &cobra.Command{
		Use:     "remove <profile>",
		Aliases: []string{"rm"},
		Short:   "Remove a profile",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			proj, err := project.Load(paths, reg)
			if err != nil {
				return err
			}
			name := args[0]
			if err := proj.RemoveProfile(name); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed profile %q\n", name)
			return nil
		},
	}
}

func newProfileUseCmd(reg *service.Registry, paths project.Paths) *cobra.Command {
	return &cobra.Command{
		Use:   "use <profile>",
		Short: "Set the active profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return useProfile(cmd, reg, paths, args[0])
		},
	}
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
