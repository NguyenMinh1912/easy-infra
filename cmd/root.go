// Package cmd defines the easy-infra command-line interface. Each command is
// intentionally thin: it parses flags and delegates to internal packages,
// where the real logic lives.
package cmd

import (
	"github.com/minhnc/easy-infra/internal/project"
	"github.com/minhnc/easy-infra/internal/service"
	"github.com/spf13/cobra"
)

// Execute builds the command tree with its dependencies and runs it. It is the
// single entrypoint called from main.
func Execute() error {
	reg := service.DefaultRegistry()
	paths := project.DefaultPaths()
	return newRootCmd(reg, paths).Execute()
}

// newRootCmd assembles the root command and its subcommands, injecting the
// service registry and project paths so commands depend on abstractions rather
// than constructing their own dependencies.
func newRootCmd(reg *service.Registry, paths project.Paths) *cobra.Command {
	root := &cobra.Command{
		Use:           "easy-infra",
		Short:         "Manage a project's local/dev infrastructure",
		Long:          "easy-infra manages local/dev infrastructure for a project through named profiles of services.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(
		newInitCmd(reg, paths),
		newProfileCmd(reg, paths),
		newUseCmd(reg, paths),
		newApplyCmd(reg, paths),
		newBackupCmd(reg, paths),
		newServeCmd(reg, paths),
	)

	return root
}
