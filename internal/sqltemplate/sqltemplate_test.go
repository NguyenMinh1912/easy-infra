package sqltemplate

import (
	"reflect"
	"testing"
)

func TestVariables(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want []string
	}{
		{"none", "SELECT 1", nil},
		{"single", "SELECT * FROM t WHERE id = {{id}}", []string{"id"}},
		{"whitespace", "WHERE a = {{ a }} AND b = {{b}}", []string{"a", "b"}},
		{"duplicates collapse, first-seen order", "{{b}} {{a}} {{b}} {{a}}", []string{"b", "a"}},
		{"underscores and digits", "{{tenant_1}}", []string{"tenant_1"}},
		{"malformed braces ignored", "{{ 1bad }} {{good}}", []string{"good"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Variables(tt.sql)
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Variables(%q) = %v, want %v", tt.sql, got, tt.want)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		tmpl    Template
		wantErr bool
	}{
		{"valid", Template{Name: "t", SQL: "SELECT 1"}, false},
		{"empty name", Template{Name: "  ", SQL: "SELECT 1"}, true},
		{"empty sql", Template{Name: "t", SQL: "  "}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.tmpl.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRender(t *testing.T) {
	t.Run("substitutes all variables", func(t *testing.T) {
		got, err := Render("WHERE since >= {{since}} AND status = {{status}}",
			map[string]string{"since": "'2026-01-01'", "status": "'open'"})
		if err != nil {
			t.Fatalf("Render: %v", err)
		}
		want := "WHERE since >= '2026-01-01' AND status = 'open'"
		if got != want {
			t.Errorf("Render = %q, want %q", got, want)
		}
	})

	t.Run("repeated variable substituted everywhere", func(t *testing.T) {
		got, err := Render("{{x}}+{{x}}", map[string]string{"x": "1"})
		if err != nil {
			t.Fatalf("Render: %v", err)
		}
		if got != "1+1" {
			t.Errorf("Render = %q, want %q", got, "1+1")
		}
	})

	t.Run("missing variable errors", func(t *testing.T) {
		_, err := Render("WHERE id = {{id}}", map[string]string{})
		if err == nil {
			t.Fatal("Render: expected error for missing variable, got nil")
		}
	})

	t.Run("no placeholders is a no-op", func(t *testing.T) {
		got, err := Render("SELECT 1", nil)
		if err != nil {
			t.Fatalf("Render: %v", err)
		}
		if got != "SELECT 1" {
			t.Errorf("Render = %q, want %q", got, "SELECT 1")
		}
	})
}
