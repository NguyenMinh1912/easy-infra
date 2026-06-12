package project

import (
	"errors"
	"testing"
)

func TestAddProfileService(t *testing.T) {
	p := newTestProject(t)
	if _, err := p.AddProfile("staging"); err != nil {
		t.Fatalf("AddProfile: %v", err)
	}

	if err := p.AddProfileService("staging", "minio"); err != nil {
		t.Fatalf("AddProfileService: %v", err)
	}

	defs, err := p.ProfileServices("staging")
	if err != nil {
		t.Fatalf("ProfileServices: %v", err)
	}
	var found bool
	for _, d := range defs {
		if d.Name == "minio" {
			found = true
		}
	}
	if !found {
		t.Errorf("minio not added to profile; got %v", defs)
	}
}

func TestAddProfileServiceErrors(t *testing.T) {
	p := newTestProject(t)
	if _, err := p.AddProfile("staging"); err != nil {
		t.Fatalf("AddProfile: %v", err)
	}

	if err := p.AddProfileService("staging", "mongodb"); !errors.Is(err, ErrUnknownService) {
		t.Errorf("AddProfileService(unknown) error = %v, want ErrUnknownService", err)
	}
	// postgres is part of the default scaffold, so adding it again conflicts.
	if err := p.AddProfileService("staging", "postgres"); !errors.Is(err, ErrServiceExists) {
		t.Errorf("AddProfileService(existing) error = %v, want ErrServiceExists", err)
	}
	if err := p.AddProfileService("nope", "minio"); err == nil {
		t.Error("AddProfileService(missing profile) error = nil, want error")
	}
}

func TestUpdateProfileService(t *testing.T) {
	p := newTestProject(t)
	if _, err := p.AddProfile("staging"); err != nil {
		t.Fatalf("AddProfile: %v", err)
	}

	cfg := map[string]any{
		"version": "15", "host": "db.example", "port": 5433, "user": "u", "database": "d",
	}
	if err := p.UpdateProfileService("staging", "postgres", cfg); err != nil {
		t.Fatalf("UpdateProfileService: %v", err)
	}

	stored, err := p.ProfileConfig("staging")
	if err != nil {
		t.Fatalf("ProfileConfig: %v", err)
	}
	if stored["postgres"]["host"] != "db.example" {
		t.Errorf("postgres host = %v, want db.example", stored["postgres"]["host"])
	}
	if stored["postgres"]["version"] != "15" {
		t.Errorf("postgres version = %v, want 15", stored["postgres"]["version"])
	}
}

func TestUpdateProfileServiceErrors(t *testing.T) {
	p := newTestProject(t)
	if _, err := p.AddProfile("staging"); err != nil {
		t.Fatalf("AddProfile: %v", err)
	}

	// minio is not in the default scaffold.
	if err := p.UpdateProfileService("staging", "minio", map[string]any{}); !errors.Is(err, ErrServiceNotDefined) {
		t.Errorf("UpdateProfileService(undefined) error = %v, want ErrServiceNotDefined", err)
	}
	// An invalid postgres config (missing required fields) is rejected.
	bad := map[string]any{"port": 99999999}
	if err := p.UpdateProfileService("staging", "postgres", bad); !errors.Is(err, ErrInvalidDefinition) {
		t.Errorf("UpdateProfileService(invalid) error = %v, want ErrInvalidDefinition", err)
	}
}

func TestRemoveProfileService(t *testing.T) {
	p := newTestProject(t)
	if _, err := p.AddProfile("staging"); err != nil {
		t.Fatalf("AddProfile: %v", err)
	}

	if err := p.RemoveProfileService("staging", "redis"); err != nil {
		t.Fatalf("RemoveProfileService: %v", err)
	}
	stored, err := p.ProfileConfig("staging")
	if err != nil {
		t.Fatalf("ProfileConfig: %v", err)
	}
	if _, ok := stored["redis"]; ok {
		t.Error("redis still present after removal")
	}
}

func TestRemoveProfileServiceErrors(t *testing.T) {
	p := newTestProject(t)
	if _, err := p.AddProfile("staging"); err != nil {
		t.Fatalf("AddProfile: %v", err)
	}

	if err := p.RemoveProfileService("staging", "minio"); !errors.Is(err, ErrServiceNotDefined) {
		t.Errorf("RemoveProfileService(undefined) error = %v, want ErrServiceNotDefined", err)
	}
	// Removing down to the last service is refused.
	if err := p.RemoveProfileService("staging", "redis"); err != nil {
		t.Fatalf("RemoveProfileService(redis): %v", err)
	}
	if err := p.RemoveProfileService("staging", "postgres"); !errors.Is(err, ErrLastService) {
		t.Errorf("RemoveProfileService(last) error = %v, want ErrLastService", err)
	}
}
