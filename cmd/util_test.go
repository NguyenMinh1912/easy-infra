package cmd

import (
	"testing"

	"github.com/minhnc/easy-infra/internal/service"
)

func TestEndpoint(t *testing.T) {
	cases := []struct {
		name string
		cfg  service.Config
		want string
	}{
		{"host and port", service.Config{"host": "db.internal", "port": 5432}, "db.internal:5432"},
		{"host only", service.Config{"host": "db.internal"}, "db.internal"},
		{"missing falls back", service.Config{}, "?"},
		{"url shows host:port", service.Config{"url": "postgres://app:secret@db.internal:5432/app"}, "db.internal:5432"},
		{"jdbc url shows host:port", service.Config{"url": "jdbc:postgresql://kot:xxxx@172.16.10.228:5432/kot?currentSchema=kot_dynamic"}, "172.16.10.228:5432"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := endpoint(tc.cfg); got != tc.want {
				t.Errorf("endpoint(%v) = %q, want %q", tc.cfg, got, tc.want)
			}
		})
	}
}
