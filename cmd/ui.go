package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/minhnc/easy-infra/internal/project"
	"github.com/minhnc/easy-infra/internal/server"
	"github.com/minhnc/easy-infra/internal/service"
	"github.com/minhnc/easy-infra/internal/workspace"
	"github.com/minhnc/easy-infra/ui"
	"github.com/spf13/cobra"
)

func newUICmd(reg *service.Registry, _ project.Paths) *cobra.Command {
	var port int
	cmd := &cobra.Command{
		Use:     "ui",
		Aliases: []string{"serve"},
		Short:   "Run the web UI and JSON API for the project",
		Long:    "Start a local HTTP server that serves the easy-infra web UI and a JSON API for inspecting the project's profiles and services.",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			uiFS, err := ui.Dist()
			if err != nil {
				return fmt.Errorf("loading embedded UI: %w", err)
			}

			ws, err := loadWorkspaces()
			if err != nil {
				return err
			}

			srv := server.New(reg, ws, uiFS)
			addr := fmt.Sprintf(":%d", port)
			fmt.Fprintf(cmd.OutOrStdout(), "easy-infra serving on http://localhost:%d\n", port)
			return srv.ListenAndServe(addr)
		},
	}
	cmd.Flags().IntVarP(&port, "port", "p", 8080, "port to listen on")
	return cmd
}

// loadWorkspaces loads the persisted workspace registry. On a fresh install
// (empty registry) it adopts the current folder as the first workspace, so the
// default behaviour — operate on the launch directory — is preserved. Once the
// registry has entries, the UI manages the list and the launch directory is no
// longer adopted automatically.
func loadWorkspaces() (*workspace.Registry, error) {
	ws, err := workspace.Load()
	if err != nil {
		return nil, err
	}
	if len(ws.Workspaces) == 0 {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		name := filepath.Base(cwd)
		if err := ws.Add(name, cwd); err != nil {
			return nil, err
		}
		_ = ws.SetActive(name)
		if err := workspace.Save(ws); err != nil {
			return nil, err
		}
	}
	return ws, nil
}
