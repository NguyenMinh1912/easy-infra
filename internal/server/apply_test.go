package server

import (
	"net/http"
	"strings"
	"testing"
)

// TestApplySnapshotNotFound rejects an unknown snapshot in the viewed profile
// before any restore runs, so a crafted id cannot escape the backups directory.
func TestApplySnapshotNotFound(t *testing.T) {
	st, reg := initProject(t, "postgres")
	srv := New(reg, st, emptyUI)

	rec := doRequest(t, srv, http.MethodPost, "/api/services/postgres/apply", `{"snapshot":"nope"}`)
	if rec.Code != http.StatusNotFound {
		t.Errorf("code = %d, want 404 (body %q)", rec.Code, rec.Body.String())
	}
}

// TestApplyUnknownSourceProfile rejects a restore whose source profile does not
// exist, rather than reading from an arbitrary backups path.
func TestApplyUnknownSourceProfile(t *testing.T) {
	st, reg := initProject(t, "postgres")
	srv := New(reg, st, emptyUI)

	rec := doRequest(t, srv, http.MethodPost, "/api/services/postgres/apply", `{"sourceProfile":"ghost"}`)
	if rec.Code != http.StatusNotFound {
		t.Errorf("code = %d, want 404 (body %q)", rec.Code, rec.Body.String())
	}
}

// TestApplySourceProfileMissingService rejects a restore whose source profile
// exists but does not define the service being applied.
func TestApplySourceProfileMissingService(t *testing.T) {
	st, reg := initProject(t, "postgres")
	addProfile(t, st, reg, "staging", "redis")
	srv := New(reg, st, emptyUI)

	rec := doRequest(t, srv, http.MethodPost, "/api/services/postgres/apply", `{"sourceProfile":"staging"}`)
	if rec.Code != http.StatusNotFound {
		t.Errorf("code = %d, want 404 (body %q)", rec.Code, rec.Body.String())
	}
}

// TestApplySnapshotNotFoundInSourceProfile validates the requested snapshot
// against the source profile's history, not the viewed profile's, and names the
// source profile in the error so the user knows where it looked.
func TestApplySnapshotNotFoundInSourceProfile(t *testing.T) {
	st, reg := initProject(t, "postgres")
	addProfile(t, st, reg, "staging", "postgres")
	srv := New(reg, st, emptyUI)

	rec := doRequest(t, srv, http.MethodPost, "/api/services/postgres/apply", `{"sourceProfile":"staging","snapshot":"nope"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("code = %d, want 404 (body %q)", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); !strings.Contains(body, "staging") {
		t.Errorf("error %q should name the source profile %q", body, "staging")
	}
}
