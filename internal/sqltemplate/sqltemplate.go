// Package sqltemplate defines a SQL template's in-memory shape, its validation,
// and the rendering that substitutes a template's variables into runnable SQL.
//
// A SQL template is a named, reusable SQL script that contains {{variable}}
// placeholders. The user fills in values for those variables and runs the
// rendered statement against a profile's queryable service. Templates are
// persisted by the store package (one row per template, scoped to a workspace);
// this package holds only the value type and the pure rules, so the storage
// layer and the HTTP layer share one definition of what a template is.
package sqltemplate

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// placeholder matches a {{ name }} variable reference. The name must start with
// a letter or underscore and contain only letters, digits, and underscores;
// surrounding whitespace inside the braces is ignored.
var placeholder = regexp.MustCompile(`\{\{\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`)

// Template is one saved SQL script. SQL is the body, which may contain
// {{variable}} placeholders; the variables are derived from the body rather
// than stored, so the body is the single source of truth.
type Template struct {
	Name        string
	Description string
	SQL         string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Variables returns the distinct variable names referenced in sql, in the order
// they first appear. A body with no placeholders yields an empty slice.
func Variables(sql string) []string {
	matches := placeholder.FindAllStringSubmatch(sql, -1)
	seen := make(map[string]bool, len(matches))
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		name := m[1]
		if seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

// Validate checks the template is coherent: it must have a name and a non-empty
// SQL body. Placeholders are validated implicitly — only well-formed {{name}}
// tokens are recognised as variables, so malformed braces are treated as
// literal SQL text and left untouched.
func (t *Template) Validate() error {
	if strings.TrimSpace(t.Name) == "" {
		return fmt.Errorf("template name is required")
	}
	if strings.TrimSpace(t.SQL) == "" {
		return fmt.Errorf("template SQL is required")
	}
	return nil
}

// Render substitutes each {{variable}} in sql with its value from vars,
// returning the runnable statement. Substitution is textual: a value is
// inserted verbatim, so a variable can stand for a literal, an identifier, or a
// whole SQL fragment. It errors, naming the variable, when sql references one
// that vars does not supply.
func Render(sql string, vars map[string]string) (string, error) {
	var missing []string
	out := placeholder.ReplaceAllStringFunc(sql, func(match string) string {
		name := placeholder.FindStringSubmatch(match)[1]
		val, ok := vars[name]
		if !ok {
			missing = append(missing, name)
			return match
		}
		return val
	})
	if len(missing) > 0 {
		return "", fmt.Errorf("missing value for variable %q", missing[0])
	}
	return out, nil
}
