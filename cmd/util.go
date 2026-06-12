package cmd

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"

	"github.com/minhnc/easy-infra/internal/service"
)

// prefixWriter prefixes every complete line written to it before forwarding to
// w. It is used to tag a service's verbose snapshot logs with the service name
// so they line up with the command's per-service output. Writes are buffered
// until a newline; lifecycle log lines are newline-terminated, so nothing is
// left dangling.
type prefixWriter struct {
	w      io.Writer
	prefix string
	buf    []byte
}

func (p *prefixWriter) Write(b []byte) (int, error) {
	p.buf = append(p.buf, b...)
	for {
		i := bytes.IndexByte(p.buf, '\n')
		if i < 0 {
			break
		}
		if _, err := io.WriteString(p.w, p.prefix); err != nil {
			return len(b), err
		}
		if _, err := p.w.Write(p.buf[:i+1]); err != nil {
			return len(b), err
		}
		p.buf = p.buf[i+1:]
	}
	return len(b), nil
}

// sortedKeys returns the keys of a service-config map in deterministic order so
// command output is stable.
func sortedKeys(m map[string]service.Config) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// endpoint renders a service's environment config as a host:port string for
// display, falling back gracefully when fields are absent. A profile may
// describe the connection as a single "url" DSN instead of discrete fields, in
// which case its host:port is shown.
func endpoint(cfg service.Config) string {
	host, _ := cfg["host"].(string)
	if host == "" {
		if u, ok := urlEndpoint(cfg); ok {
			return u
		}
		host = "?"
	}
	if port, ok := cfg["port"]; ok {
		return fmt.Sprintf("%s:%v", host, port)
	}
	return host
}

// urlEndpoint extracts a "host:port" (or just host) from a "url" DSN field for
// display. JDBC-style URLs carry a leading "jdbc:" prefix that url.Parse does
// not understand, so it is stripped first. Returns false when no usable host is
// present.
func urlEndpoint(cfg service.Config) (string, bool) {
	raw, _ := cfg["url"].(string)
	if raw == "" {
		return "", false
	}
	u, err := url.Parse(strings.TrimPrefix(raw, "jdbc:"))
	if err != nil || u.Host == "" {
		return "", false
	}
	return u.Host, true
}
