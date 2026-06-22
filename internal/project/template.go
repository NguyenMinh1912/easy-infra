package project

import (
	"errors"
	"fmt"

	"github.com/minhnc/easy-infra/internal/sqltemplate"
	"github.com/minhnc/easy-infra/internal/store"
)

// Templates lists the workspace's SQL templates, sorted by name.
func (p *Project) Templates() ([]sqltemplate.Template, error) {
	return p.Store.ListTemplates(p.Workspace.ID)
}

// Template returns the named template, reporting a missing one as an actionable
// error.
func (p *Project) Template(name string) (sqltemplate.Template, error) {
	t, err := p.Store.GetTemplate(p.Workspace.ID, name)
	if err != nil {
		if errors.Is(err, store.ErrTemplateNotFound) {
			return sqltemplate.Template{}, fmt.Errorf("template %q does not exist", name)
		}
		return sqltemplate.Template{}, err
	}
	return t, nil
}

// CreateTemplate validates and saves a new template. A duplicate name is
// reported as an actionable error.
func (p *Project) CreateTemplate(t sqltemplate.Template) error {
	if err := t.Validate(); err != nil {
		return err
	}
	if err := p.Store.CreateTemplate(p.Workspace.ID, t); err != nil {
		if errors.Is(err, store.ErrTemplateExists) {
			return fmt.Errorf("template %q already exists", t.Name)
		}
		return err
	}
	return nil
}

// UpdateTemplate validates and saves the named template's description and SQL.
// A missing template is reported as an actionable error.
func (p *Project) UpdateTemplate(name string, t sqltemplate.Template) error {
	// Validate against the target name, not whatever the body carries, so a
	// body with an empty Name still passes the name check.
	check := t
	check.Name = name
	if err := check.Validate(); err != nil {
		return err
	}
	if err := p.Store.UpdateTemplate(p.Workspace.ID, name, t); err != nil {
		if errors.Is(err, store.ErrTemplateNotFound) {
			return fmt.Errorf("template %q does not exist", name)
		}
		return err
	}
	return nil
}

// RemoveTemplate deletes the named template, reporting a missing one as an
// actionable error.
func (p *Project) RemoveTemplate(name string) error {
	if err := p.Store.RemoveTemplate(p.Workspace.ID, name); err != nil {
		if errors.Is(err, store.ErrTemplateNotFound) {
			return fmt.Errorf("template %q does not exist", name)
		}
		return err
	}
	return nil
}

// RenderTemplate loads the named template and substitutes vars into its SQL,
// returning the runnable statement. It errors when the template is missing or a
// referenced variable has no value.
func (p *Project) RenderTemplate(name string, vars map[string]string) (string, error) {
	t, err := p.Template(name)
	if err != nil {
		return "", err
	}
	return sqltemplate.Render(t.SQL, vars)
}
