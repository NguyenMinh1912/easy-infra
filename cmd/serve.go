package cmd

import (
	"fmt"

	"github.com/minhnc/easy-infra/internal/project"
	"github.com/minhnc/easy-infra/internal/server"
	"github.com/minhnc/easy-infra/internal/service"
	"github.com/minhnc/easy-infra/ui"
	"github.com/spf13/cobra"
)

func newServeCmd(reg *service.Registry, paths project.Paths) *cobra.Command {
	var port int
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve the web UI and JSON API for the project",
		Long:  "Start a local HTTP server that serves the easy-infra web UI and a JSON API for inspecting the project's profiles and services.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			uiFS, err := ui.Dist()
			if err != nil {
				return fmt.Errorf("loading embedded UI: %w", err)
			}
			srv := server.New(reg, paths, uiFS)
			addr := fmt.Sprintf(":%d", port)
			fmt.Fprintf(cmd.OutOrStdout(), "easy-infra serving on http://localhost:%d\n", port)
			return srv.ListenAndServe(addr)
		},
	}
	cmd.Flags().IntVarP(&port, "port", "p", 8080, "port to listen on")
	return cmd
}
