package cmd

import (
	"github.com/minhnc/easy-infra/internal/service"
	"github.com/spf13/cobra"
)

// newServeCmd exposes `easy-infra serve` (alias `ui`) as an explicit way to
// start the app, mirroring the root command's default behaviour. It shares the
// root's --port flag.
func newServeCmd(reg *service.Registry, port *int) *cobra.Command {
	return &cobra.Command{
		Use:     "serve",
		Aliases: []string{"ui"},
		Short:   "Run the web UI and JSON API",
		Long:    "Start a local HTTP server that serves the easy-infra web UI and JSON API. This is also what running easy-infra with no subcommand does.",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runServe(cmd, reg, *port)
		},
	}
}
