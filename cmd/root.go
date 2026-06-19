// Package cmd defines the easy-infra entrypoint. easy-infra is an app: the
// binary opens the central store and serves the web UI + JSON API, where the
// user manages workspaces, profiles, and services. There is no separate CLI for
// those operations — running easy-infra launches the app.
package cmd

import (
	"fmt"

	"github.com/minhnc/easy-infra/internal/server"
	"github.com/minhnc/easy-infra/internal/service"
	"github.com/minhnc/easy-infra/internal/store"
	"github.com/minhnc/easy-infra/ui"
	"github.com/spf13/cobra"
)

// version is the build version, stamped at release time via ldflags
// (-X github.com/minhnc/easy-infra/cmd.version=<tag>). It powers
// `easy-infra --version`; unstamped dev builds report "dev".
var version = "dev"

// Execute builds the command and runs it. It is the single entrypoint called
// from main.
func Execute() error {
	return newRootCmd(service.DefaultRegistry()).Execute()
}

// newRootCmd assembles the root command. Running it (with no subcommand) starts
// the server; the injected service registry is the only dependency it needs,
// since all other state lives in the store opened at run time.
func newRootCmd(reg *service.Registry) *cobra.Command {
	var port int
	root := &cobra.Command{
		Use:           "easy-infra",
		Short:         "Manage a project's local/dev infrastructure",
		Long:          "easy-infra serves a web UI and JSON API for managing local/dev infrastructure through named profiles of services. All data is stored in a single local database.",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runServe(cmd, reg, port)
		},
	}
	root.Flags().IntVarP(&port, "port", "p", 8080, "port to listen on")
	// `serve`/`ui` remain as explicit aliases for discoverability and to keep
	// existing install scripts/launchers working.
	root.AddCommand(newServeCmd(reg, &port))
	return root
}

// runServe opens the store and serves the app until the process is stopped.
func runServe(cmd *cobra.Command, reg *service.Registry, port int) error {
	uiFS, err := ui.Dist()
	if err != nil {
		return fmt.Errorf("loading embedded UI: %w", err)
	}
	st, err := store.Open()
	if err != nil {
		return err
	}
	defer st.Close()

	srv := server.New(reg, st, uiFS)
	addr := fmt.Sprintf(":%d", port)
	fmt.Fprintf(cmd.OutOrStdout(), "easy-infra serving on http://localhost:%d\n", port)
	return srv.ListenAndServe(addr)
}
