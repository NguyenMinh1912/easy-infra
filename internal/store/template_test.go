package store

import (
	"errors"
	"testing"

	"github.com/minhnc/easy-infra/internal/sqltemplate"
)

func TestTemplateCRUD(t *testing.T) {
	s := open(t)
	ws, err := s.CreateWorkspace("app")
	if err != nil {
		t.Fatalf("CreateWorkspace: %v", err)
	}

	// Empty to start.
	list, err := s.ListTemplates(ws.ID)
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected no templates, got %d", len(list))
	}

	// Create.
	tmpl := sqltemplate.Template{Name: "active-users", Description: "since a date", SQL: "SELECT * FROM users WHERE last_seen >= {{since}}"}
	if err := s.CreateTemplate(ws.ID, tmpl); err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}

	// Get.
	got, err := s.GetTemplate(ws.ID, "active-users")
	if err != nil {
		t.Fatalf("GetTemplate: %v", err)
	}
	if got.Name != tmpl.Name || got.Description != tmpl.Description || got.SQL != tmpl.SQL {
		t.Fatalf("GetTemplate = %+v, want %+v", got, tmpl)
	}
	if got.CreatedAt.IsZero() || got.UpdatedAt.IsZero() {
		t.Errorf("timestamps not set: %+v", got)
	}

	// Update.
	if err := s.UpdateTemplate(ws.ID, "active-users", sqltemplate.Template{Description: "updated", SQL: "SELECT 1"}); err != nil {
		t.Fatalf("UpdateTemplate: %v", err)
	}
	got, err = s.GetTemplate(ws.ID, "active-users")
	if err != nil {
		t.Fatalf("GetTemplate after update: %v", err)
	}
	if got.Description != "updated" || got.SQL != "SELECT 1" {
		t.Fatalf("after update = %+v, want description=updated sql=SELECT 1", got)
	}

	// Remove.
	if err := s.RemoveTemplate(ws.ID, "active-users"); err != nil {
		t.Fatalf("RemoveTemplate: %v", err)
	}
	if _, err := s.GetTemplate(ws.ID, "active-users"); !errors.Is(err, ErrTemplateNotFound) {
		t.Fatalf("GetTemplate after remove err = %v, want ErrTemplateNotFound", err)
	}
}

func TestTemplateDuplicateName(t *testing.T) {
	s := open(t)
	ws, _ := s.CreateWorkspace("app")
	tmpl := sqltemplate.Template{Name: "t", SQL: "SELECT 1"}
	if err := s.CreateTemplate(ws.ID, tmpl); err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}
	if err := s.CreateTemplate(ws.ID, tmpl); !errors.Is(err, ErrTemplateExists) {
		t.Fatalf("duplicate CreateTemplate err = %v, want ErrTemplateExists", err)
	}
}

func TestTemplateMissingErrors(t *testing.T) {
	s := open(t)
	ws, _ := s.CreateWorkspace("app")
	if err := s.UpdateTemplate(ws.ID, "nope", sqltemplate.Template{SQL: "x"}); !errors.Is(err, ErrTemplateNotFound) {
		t.Errorf("UpdateTemplate missing err = %v, want ErrTemplateNotFound", err)
	}
	if err := s.RemoveTemplate(ws.ID, "nope"); !errors.Is(err, ErrTemplateNotFound) {
		t.Errorf("RemoveTemplate missing err = %v, want ErrTemplateNotFound", err)
	}
}

func TestTemplateCascadesWithWorkspace(t *testing.T) {
	s := open(t)
	ws, _ := s.CreateWorkspace("app")
	if err := s.CreateTemplate(ws.ID, sqltemplate.Template{Name: "t", SQL: "SELECT 1"}); err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}
	if err := s.RemoveWorkspace(ws.ID); err != nil {
		t.Fatalf("RemoveWorkspace: %v", err)
	}
	// The template row is gone with its workspace; a fresh workspace reusing the
	// id (none here) would not see it. Listing on the removed id yields nothing.
	list, err := s.ListTemplates(ws.ID)
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected templates to cascade away, got %d", len(list))
	}
}
