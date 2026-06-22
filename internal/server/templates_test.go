package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/minhnc/easy-infra/internal/service"
)

func decodeTemplate(t *testing.T, body []byte) templateResponse {
	t.Helper()
	var got templateResponse
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode template: %v (body %q)", err, body)
	}
	return got
}

func TestListTemplatesNotInitialized(t *testing.T) {
	srv := New(service.DefaultRegistry(), newStore(t), emptyUI)
	rec := doRequest(t, srv, http.MethodGet, "/api/templates", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	var got []templateSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("templates = %+v, want empty", got)
	}
}

func TestTemplateCRUDFlow(t *testing.T) {
	st, reg := initProject(t, "postgres")
	srv := New(reg, st, emptyUI)

	// Create.
	rec := doJSON(t, srv, http.MethodPost, "/api/templates",
		templateRequest{Name: "active-users", Description: "since a date", SQL: "SELECT * FROM users WHERE last_seen >= {{since}}"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201 (body %q)", rec.Code, rec.Body.String())
	}
	got := decodeTemplate(t, rec.Body.Bytes())
	if len(got.Variables) != 1 || got.Variables[0] != "since" {
		t.Fatalf("variables = %v, want [since]", got.Variables)
	}

	// List.
	rec = doRequest(t, srv, http.MethodGet, "/api/templates", "")
	var list []templateSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list) != 1 || list[0].Name != "active-users" {
		t.Fatalf("list = %+v, want one active-users", list)
	}

	// Get.
	rec = doRequest(t, srv, http.MethodGet, "/api/templates/active-users", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want 200", rec.Code)
	}
	got = decodeTemplate(t, rec.Body.Bytes())
	if got.SQL == "" {
		t.Fatal("get returned empty SQL")
	}

	// Update.
	rec = doJSON(t, srv, http.MethodPut, "/api/templates/active-users",
		templateRequest{Description: "updated", SQL: "SELECT 1"})
	if rec.Code != http.StatusOK {
		t.Fatalf("update status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	got = decodeTemplate(t, rec.Body.Bytes())
	if got.Description != "updated" || got.SQL != "SELECT 1" {
		t.Fatalf("after update = %+v", got)
	}

	// Delete.
	rec = doRequest(t, srv, http.MethodDelete, "/api/templates/active-users", "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", rec.Code)
	}
	rec = doRequest(t, srv, http.MethodGet, "/api/templates/active-users", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("get after delete status = %d, want 404", rec.Code)
	}
}

func TestCreateTemplateInvalidName(t *testing.T) {
	st, reg := initProject(t, "postgres")
	srv := New(reg, st, emptyUI)
	rec := doJSON(t, srv, http.MethodPost, "/api/templates",
		templateRequest{Name: "bad name", SQL: "SELECT 1"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestCreateTemplateDuplicate(t *testing.T) {
	st, reg := initProject(t, "postgres")
	srv := New(reg, st, emptyUI)
	body := templateRequest{Name: "t", SQL: "SELECT 1"}
	if rec := doJSON(t, srv, http.MethodPost, "/api/templates", body); rec.Code != http.StatusCreated {
		t.Fatalf("first create status = %d, want 201", rec.Code)
	}
	if rec := doJSON(t, srv, http.MethodPost, "/api/templates", body); rec.Code != http.StatusConflict {
		t.Fatalf("duplicate create status = %d, want 409", rec.Code)
	}
}

func TestRunTemplateMissingVariable(t *testing.T) {
	st, reg := initProject(t, "postgres")
	srv := New(reg, st, emptyUI)
	if rec := doJSON(t, srv, http.MethodPost, "/api/templates",
		templateRequest{Name: "t", SQL: "SELECT {{n}}"}); rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", rec.Code)
	}
	// No value for {{n}} → 400 before any database work.
	rec := doJSON(t, srv, http.MethodPost, "/api/templates/t/run",
		templateRunRequest{Profile: "default", Service: "postgres", Variables: map[string]string{}})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("run status = %d, want 400 (body %q)", rec.Code, rec.Body.String())
	}
}

func TestRunTemplateUnknownProfile(t *testing.T) {
	st, reg := initProject(t, "postgres")
	srv := New(reg, st, emptyUI)
	if rec := doJSON(t, srv, http.MethodPost, "/api/templates",
		templateRequest{Name: "t", SQL: "SELECT 1"}); rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", rec.Code)
	}
	rec := doJSON(t, srv, http.MethodPost, "/api/templates/t/run",
		templateRunRequest{Profile: "nope", Service: "postgres", Variables: map[string]string{}})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("run status = %d, want 404 (body %q)", rec.Code, rec.Body.String())
	}
}
