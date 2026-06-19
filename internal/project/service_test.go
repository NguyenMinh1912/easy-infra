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

	id, err := p.AddProfileService("staging", "minio", "", nil)
	if err != nil {
		t.Fatalf("AddProfileService: %v", err)
	}
	if id != "minio" {
		t.Errorf("first minio id = %q, want %q", id, "minio")
	}

	defs, err := p.ProfileServices("staging")
	if err != nil {
		t.Fatalf("ProfileServices: %v", err)
	}
	var found bool
	for _, d := range defs {
		if d.ID == "minio" && d.Type == "minio" {
			found = true
		}
	}
	if !found {
		t.Errorf("minio not added to profile; got %v", defs)
	}
}

func TestAddProfileServiceMultipleSameType(t *testing.T) {
	p := newTestProject(t)
	if _, err := p.AddProfile("staging"); err != nil {
		t.Fatalf("AddProfile: %v", err)
	}
	seedStarterServices(t, p, "staging")

	// postgres is already defined, so a second instance gets a distinct,
	// suffixed id rather than being rejected.
	id, err := p.AddProfileService("staging", "postgres", "Analytics", nil)
	if err != nil {
		t.Fatalf("AddProfileService(second postgres): %v", err)
	}
	if id != "postgres-2" {
		t.Errorf("second postgres id = %q, want %q", id, "postgres-2")
	}

	// A third instance continues the sequence.
	id3, err := p.AddProfileService("staging", "postgres", "", nil)
	if err != nil {
		t.Fatalf("AddProfileService(third postgres): %v", err)
	}
	if id3 != "postgres-3" {
		t.Errorf("third postgres id = %q, want %q", id3, "postgres-3")
	}

	defs, err := p.ProfileServices("staging")
	if err != nil {
		t.Fatalf("ProfileServices: %v", err)
	}
	count := 0
	var analyticsName string
	for _, d := range defs {
		if d.Type == "postgres" {
			count++
		}
		if d.ID == "postgres-2" {
			analyticsName = d.Name
		}
	}
	if count != 3 {
		t.Errorf("postgres instance count = %d, want 3", count)
	}
	if analyticsName != "Analytics" {
		t.Errorf("postgres-2 name = %q, want %q", analyticsName, "Analytics")
	}
}

func TestAddProfileServiceErrors(t *testing.T) {
	p := newTestProject(t)
	if _, err := p.AddProfile("staging"); err != nil {
		t.Fatalf("AddProfile: %v", err)
	}

	if _, err := p.AddProfileService("staging", "mongodb", "", nil); !errors.Is(err, ErrUnknownService) {
		t.Errorf("AddProfileService(unknown) error = %v, want ErrUnknownService", err)
	}
	if _, err := p.AddProfileService("nope", "minio", "", nil); err == nil {
		t.Error("AddProfileService(missing profile) error = nil, want error")
	}
}

func TestUpdateProfileService(t *testing.T) {
	p := newTestProject(t)
	if _, err := p.AddProfile("staging"); err != nil {
		t.Fatalf("AddProfile: %v", err)
	}
	seedStarterServices(t, p, "staging")

	cfg := map[string]any{
		"version": "15", "host": "db.example", "port": 5433, "user": "u", "database": "d",
	}
	if err := p.UpdateProfileService("staging", "postgres", "", cfg); err != nil {
		t.Fatalf("UpdateProfileService: %v", err)
	}

	stored, err := p.ProfileConfig("staging")
	if err != nil {
		t.Fatalf("ProfileConfig: %v", err)
	}
	if stored["postgres"].Config["host"] != "db.example" {
		t.Errorf("postgres host = %v, want db.example", stored["postgres"].Config["host"])
	}
	if stored["postgres"].Config["version"] != "15" {
		t.Errorf("postgres version = %v, want 15", stored["postgres"].Config["version"])
	}
}

func TestUpdateProfileServiceRename(t *testing.T) {
	p := newTestProject(t)
	if _, err := p.AddProfile("staging"); err != nil {
		t.Fatalf("AddProfile: %v", err)
	}
	seedStarterServices(t, p, "staging")

	cfg := map[string]any{
		"version": "16", "host": "localhost", "port": 5432, "user": "u", "database": "d",
	}
	if err := p.UpdateProfileService("staging", "postgres", "Primary DB", cfg); err != nil {
		t.Fatalf("UpdateProfileService(rename): %v", err)
	}

	defs, err := p.ProfileServices("staging")
	if err != nil {
		t.Fatalf("ProfileServices: %v", err)
	}
	var name string
	for _, d := range defs {
		if d.ID == "postgres" {
			name = d.Name
		}
	}
	if name != "Primary DB" {
		t.Errorf("postgres name = %q, want %q", name, "Primary DB")
	}
}

func TestUpdateProfileServiceErrors(t *testing.T) {
	p := newTestProject(t)
	if _, err := p.AddProfile("staging"); err != nil {
		t.Fatalf("AddProfile: %v", err)
	}
	seedStarterServices(t, p, "staging")

	// minio is not defined on the profile.
	if err := p.UpdateProfileService("staging", "minio", "", map[string]any{}); !errors.Is(err, ErrServiceNotDefined) {
		t.Errorf("UpdateProfileService(undefined) error = %v, want ErrServiceNotDefined", err)
	}
	// An invalid postgres config (missing required fields) is rejected.
	bad := map[string]any{"port": 99999999}
	if err := p.UpdateProfileService("staging", "postgres", "", bad); !errors.Is(err, ErrInvalidDefinition) {
		t.Errorf("UpdateProfileService(invalid) error = %v, want ErrInvalidDefinition", err)
	}
}

func TestRemoveProfileService(t *testing.T) {
	p := newTestProject(t)
	if _, err := p.AddProfile("staging"); err != nil {
		t.Fatalf("AddProfile: %v", err)
	}
	seedStarterServices(t, p, "staging")

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
	seedStarterServices(t, p, "staging")

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
