package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/minhnc/easy-infra/internal/backup"
	"github.com/minhnc/easy-infra/internal/project"
	"github.com/minhnc/easy-infra/internal/service"
)

// handleStartFork forks a service from the active (source) profile to a local
// container in the background, returning the (new or already-running) session as
// JSON. The body selects which backup version to seed the local instance from;
// an empty snapshot means "create a fresh backup of the source first, then fork
// from it". The fork:
//
//  1. (optionally) backs up the source service into a new snapshot,
//  2. writes the localised service env into the conventional "local" profile,
//  3. launches the local container (same config as the source), and
//  4. restores the chosen snapshot into it.
//
// Steps 2 happens synchronously so a bad profile is reported as an error before
// a session is created; the rest stream their progress through the session log.
func (s *Server) handleStartFork(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	var body struct {
		Snapshot string `json:"snapshot"`
		// Port overrides the local container's published port; 0 keeps the
		// source profile's port.
		Port int `json:"port"`
	}
	if r.Body != nil {
		// An empty body is allowed and means "create a new backup"; only a
		// malformed one errs.
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err != io.EOF {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
			return
		}
	}

	proj, err := project.Load(s.paths, s.reg)
	if err != nil {
		s.writeProjectError(w, err)
		return
	}
	sourceProfile, prof, err := proj.ActiveProfile()
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	env, ok := prof.Services[name]
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("service %q is not defined in profile %q", name, sourceProfile))
		return
	}
	svc, ok := s.reg.Get(name)
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("unknown service %q", name))
		return
	}
	def := proj.Config.Services[name]
	store, err := s.backups.Store()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Services without container provisioning still get a session so the UI can
	// surface "not supported yet" through the same progress dialog.
	prov, provOK := svc.(service.Provisioner)
	if !provOK {
		run := func(_ context.Context, lw io.Writer) error {
			fmt.Fprintf(lw, "forking %q to a local container is not supported yet\n", name)
			return service.ErrNotImplemented
		}
		sess, err := s.backups.Start(store, name, project.LocalProfile, backup.KindFork, "", nil, run)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusAccepted, toSessionJSON(sess))
		return
	}

	// Validate a requested snapshot against the source profile's known list, so a
	// crafted id cannot escape the backups directory.
	if body.Snapshot != "" {
		ids, err := service.ListSnapshots(sourceProfile)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !contains(ids, body.Snapshot) {
			writeError(w, http.StatusNotFound, fmt.Sprintf("snapshot %q not found for profile %q", body.Snapshot, sourceProfile))
			return
		}
	}

	// Derive the local env and persist the local profile up front, so an invalid
	// config fails the request rather than a background session.
	localEnv, err := prov.LocalEnv(env)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	// Let the caller publish the local container on a different port than the
	// source (e.g. to avoid clashing with a port already in use locally). 0
	// keeps the derived source port.
	if body.Port != 0 {
		if body.Port < 1 || body.Port > 65535 {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("port must be between 1 and 65535, got %d", body.Port))
			return
		}
		localEnv["port"] = body.Port
	}
	localProfile, err := proj.ForkLocalProfile(sourceProfile, name, localEnv)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// When no version is chosen, take a fresh backup of the source first. Reserve
	// its snapshot folder now so the session records the version it produced.
	createNew := body.Snapshot == ""
	snapshot := body.Snapshot
	var newBackupDir string
	if createNew {
		newBackupDir = service.NewSnapshotDir(sourceProfile)
		snapshot = filepath.Base(newBackupDir)
	}

	run := func(ctx context.Context, lw io.Writer) error {
		if createNew {
			fmt.Fprintf(lw, "No backup selected; snapshotting %q from profile %q into %s\n", name, sourceProfile, newBackupDir)
			if err := svc.Backup(ctx, service.Spec{
				Profile:    sourceProfile,
				Definition: def,
				Env:        env,
				BackupDir:  newBackupDir,
				Log:        lw,
			}); err != nil {
				return err
			}
		}

		fmt.Fprintf(lw, "Forking %q to local profile %q\n", name, localProfile)
		if err := prov.Provision(ctx, service.Spec{
			Profile:    localProfile,
			Definition: def,
			Env:        localEnv,
			Log:        lw,
		}); err != nil {
			return err
		}

		fmt.Fprintf(lw, "Restoring snapshot %s into the local %q\n", snapshot, name)
		// Profile is the SOURCE so the snapshot is located under it; Env is the
		// LOCAL container so the restore lands in the fork.
		return svc.Apply(ctx, service.Spec{
			Profile:    sourceProfile,
			Definition: def,
			Env:        localEnv,
			Snapshot:   snapshot,
			Log:        lw,
		})
	}

	// A cancelled/failed fresh backup may have written a partial snapshot under
	// the source profile; drop it. An existing snapshot is left untouched.
	var cleanup func()
	if createNew {
		cleanup = func() { _ = os.RemoveAll(newBackupDir) }
	}

	sess, err := s.backups.Start(store, name, localProfile, backup.KindFork, snapshot, cleanup, run)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, toSessionJSON(sess))
}
