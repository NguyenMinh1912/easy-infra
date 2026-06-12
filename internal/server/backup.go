package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/minhnc/easy-infra/internal/project"
	"github.com/minhnc/easy-infra/internal/service"
)

// doneEvent is the JSON payload of the terminal "done" SSE event. Status is
// "ok" for a completed backup and "unsupported" for a service whose backup is
// not wired up yet; Snapshot names the snapshot folder on success.
type doneEvent struct {
	Status   string `json:"status"`
	Snapshot string `json:"snapshot,omitempty"`
}

// errorEvent is the JSON payload of the terminal "error" SSE event.
type errorEvent struct {
	Error string `json:"error"`
}

// handleBackupService snapshots a single service for the active profile,
// streaming the service's verbose progress to the client as Server-Sent Events
// so the UI can show a live log. The whole-profile snapshot lives in the CLI
// (`easy-infra backup snapshot`); here the action is per-service, matching where
// it is offered in the UI.
//
// Failures that occur before streaming starts (no project, no active profile,
// unknown service) are returned as the usual JSON error envelope with a status
// code. Once the stream opens, the outcome is carried by a terminal "done" or
// "error" event instead, since the headers are already committed to 200.
func (s *Server) handleBackupService(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	proj, err := project.Load(s.paths, s.reg)
	if err != nil {
		s.writeProjectError(w, err)
		return
	}
	profileName, prof, err := proj.ActiveProfile()
	if err != nil {
		// No active profile (or an invalid one) is resolved with `use`.
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	env, ok := prof.Services[name]
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("service %q is not defined in profile %q", name, profileName))
		return
	}
	svc, ok := s.reg.Get(name)
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("unknown service %q", name))
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming is not supported by this server")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()
	sse := &sseWriter{w: w, flusher: flusher}

	// One fresh snapshot folder for this service's artifact.
	dir := service.NewSnapshotDir(profileName)
	sse.log(fmt.Sprintf("Backing up %q (profile %q) into %s", name, profileName, dir))

	spec := service.Spec{
		Profile:    profileName,
		Definition: proj.Config.Services[name],
		Env:        env,
		BackupDir:  dir,
		Log:        sse, // verbose lines stream out as "log" events
	}

	switch err := svc.Backup(r.Context(), spec); {
	case errors.Is(err, service.ErrNotImplemented):
		sse.log(fmt.Sprintf("backup is not supported for %q yet", name))
		sse.json("done", doneEvent{Status: "unsupported"})
	case err != nil:
		sse.json("error", errorEvent{Error: err.Error()})
	default:
		sse.json("done", doneEvent{Status: "ok", Snapshot: filepath.Base(dir)})
	}
}

// sseWriter streams Server-Sent Events to an HTTP response. As an io.Writer it
// is wired to Spec.Log: writes are buffered until a newline and each complete
// line is emitted as a "log" event, so a service's verbose progress reaches the
// client line by line. Terminal "done"/"error" events are sent via json.
type sseWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
	buf     []byte
}

// Write implements io.Writer, emitting each complete line as a "log" event.
// Lifecycle log lines are newline-terminated, so nothing is left dangling.
func (s *sseWriter) Write(b []byte) (int, error) {
	s.buf = append(s.buf, b...)
	for {
		i := bytes.IndexByte(s.buf, '\n')
		if i < 0 {
			break
		}
		line := string(s.buf[:i])
		s.buf = s.buf[i+1:]
		s.log(line)
	}
	return len(b), nil
}

// log emits a single line as a "log" event.
func (s *sseWriter) log(line string) {
	s.frame("log", line)
}

// json emits an event whose data is the compact JSON encoding of v.
func (s *sseWriter) json(event string, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	s.frame(event, string(data))
}

// frame writes one SSE event and flushes it. data must be a single line; log
// lines (split on newline) and compact JSON both satisfy that.
func (s *sseWriter) frame(event, data string) {
	fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", event, data)
	s.flusher.Flush()
}
