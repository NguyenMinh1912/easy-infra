package cmd

import (
	"errors"
	"fmt"

	"github.com/minhnc/easy-infra/internal/profile"
	"github.com/minhnc/easy-infra/internal/project"
	"github.com/minhnc/easy-infra/internal/service"
	"github.com/spf13/cobra"
)

func newBackupCmd(reg *service.Registry, paths project.Paths) *cobra.Command {
	backup := &cobra.Command{
		Use:   "backup",
		Short: "Manage backup snapshots",
		Long:  "Snapshot the services in a profile into a versioned backup folder, and list the snapshots taken so far.",
	}
	backup.AddCommand(
		newBackupSnapshotCmd(reg, paths),
		newBackupListCmd(reg, paths),
	)
	return backup
}

func newBackupSnapshotCmd(reg *service.Registry, paths project.Paths) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot [profile]",
		Short: "Snapshot a profile's services into a new backup folder",
		Long:  "Back up every service in a profile into a single timestamped snapshot folder. Defaults to the active profile; pass a profile name to snapshot a specific one.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			proj, err := project.Load(paths, reg)
			if err != nil {
				return err
			}
			name, prof, err := resolveProfile(proj, args)
			if err != nil {
				return err
			}

			// One shared folder per snapshot, so every service is captured together.
			dir := service.NewSnapshotDir(name)
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Snapshotting profile %q into %s:\n", name, dir)
			ctx := cmd.Context()
			for _, svcName := range sortedKeys(prof.Services) {
				svc, ok := reg.Get(svcName)
				if !ok {
					return fmt.Errorf("unknown service %q", svcName)
				}
				spec := service.Spec{
					Profile:    name,
					Definition: proj.Config.Services[svcName],
					Env:        prof.Services[svcName],
					BackupDir:  dir,
				}
				// In verbose mode, stream the service's own progress lines, tagged
				// with its name so they line up with the per-service summary below.
				if verbose {
					spec.Log = &prefixWriter{w: out, prefix: fmt.Sprintf("  - %s: ", svcName)}
				}
				switch err := svc.Backup(ctx, spec); {
				case errors.Is(err, service.ErrNotImplemented):
					fmt.Fprintf(out, "  - %s: would back up from %s\n", svcName, endpoint(spec.Env))
				case err != nil:
					return fmt.Errorf("backing up %s: %w", svcName, err)
				default:
					fmt.Fprintf(out, "  - %s: backed up from %s\n", svcName, endpoint(spec.Env))
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolP("verbose", "v", false, "stream each service's snapshot activity as it runs")
	return cmd
}

func newBackupListCmd(reg *service.Registry, paths project.Paths) *cobra.Command {
	return &cobra.Command{
		Use:   "list [profile]",
		Short: "List backup snapshots",
		Long:  "List the snapshots taken for each profile. Pass a profile name to list only that profile's snapshots.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			proj, err := project.Load(paths, reg)
			if err != nil {
				return err
			}

			var profiles []string
			if len(args) == 1 {
				profiles = []string{args[0]}
			} else if profiles, err = proj.Profiles(); err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			found := false
			for _, name := range profiles {
				ids, err := service.ListSnapshots(name)
				if err != nil {
					return err
				}
				if len(ids) == 0 {
					continue
				}
				found = true
				fmt.Fprintf(out, "%s:\n", name)
				// Newest first: snapshot ids are sortable timestamps.
				for i := len(ids) - 1; i >= 0; i-- {
					fmt.Fprintf(out, "  %s\n", ids[i])
				}
			}
			if !found {
				fmt.Fprintln(out, "no backups found")
			}
			return nil
		},
	}
}

// resolveProfile picks the profile a backup command operates on: the one named
// in args, or the active profile when none is given. It loads and validates the
// profile so callers get an actionable error for an unknown or invalid one.
func resolveProfile(proj *project.Project, args []string) (string, *profile.Profile, error) {
	if len(args) == 1 {
		name := args[0]
		prof, err := proj.LoadProfile(name)
		if err != nil {
			return "", nil, err
		}
		return name, prof, nil
	}
	return proj.ActiveProfile()
}
