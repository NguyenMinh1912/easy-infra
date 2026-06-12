package service

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Backups are organised as versioned snapshots, kept per service so each
// service has an independent history and a snapshot folder's owning service is
// obvious from its path:
//
//	.easy-infra/backups/<profile>/<service>/<snapshot>/...
//
// A profile-wide snapshot (e.g. `easy-infra backup snapshot`) reuses one
// timestamp across the services it captures, so the same id lines up across
// each service's folder. Snapshot ids are sortable UTC timestamps, so the
// lexically greatest is the newest.

// BackupsDir is the directory holding a profile's per-service backup folders,
// relative to the project root (matching how config/state paths are resolved).
func BackupsDir(profile string) string {
	return filepath.Join(".easy-infra", "backups", profile)
}

// ServiceBackupsDir is the directory holding one service's backup snapshots
// within a profile.
func ServiceBackupsDir(profile, svc string) string {
	return filepath.Join(BackupsDir(profile), svc)
}

// NewSnapshotDir returns the path for a fresh snapshot of svc in profile, named
// by a sortable UTC timestamp.
func NewSnapshotDir(profile, svc string) string {
	return SnapshotDir(profile, svc, backupStamp())
}

// ListSnapshots returns svc's snapshot ids (the timestamp folder names) within
// profile in sorted order — oldest first, newest last. A missing backups dir
// yields an empty list rather than an error.
func ListSnapshots(profile, svc string) ([]string, error) {
	dir := ServiceBackupsDir(profile, svc)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading backups %s: %w", dir, err)
	}
	var ids []string
	for _, e := range entries {
		if e.IsDir() {
			ids = append(ids, e.Name())
		}
	}
	sort.Strings(ids)
	return ids, nil
}

// BackedUpServices returns the names of services that have at least one snapshot
// folder under profile, sorted. A missing backups dir yields an empty list.
func BackedUpServices(profile string) ([]string, error) {
	entries, err := os.ReadDir(BackupsDir(profile))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading backups %s: %w", BackupsDir(profile), err)
	}
	var svcs []string
	for _, e := range entries {
		if e.IsDir() {
			svcs = append(svcs, e.Name())
		}
	}
	sort.Strings(svcs)
	return svcs, nil
}

// SnapshotDir returns the path of svc's snapshot folder named id under profile.
// The id is a folder name produced by NewSnapshotDir and listed by
// ListSnapshots; callers validate it exists before trusting client input.
func SnapshotDir(profile, svc, id string) string {
	return filepath.Join(ServiceBackupsDir(profile, svc), id)
}

// latestSnapshotDir returns the path of svc's newest snapshot folder in profile,
// or an empty string when none exist.
func latestSnapshotDir(profile, svc string) (string, error) {
	ids, err := ListSnapshots(profile, svc)
	if err != nil || len(ids) == 0 {
		return "", err
	}
	return SnapshotDir(profile, svc, ids[len(ids)-1]), nil
}

// BackupStamp returns a sortable, filesystem-safe UTC timestamp suitable for
// naming a snapshot. The command layer uses it to share one id across the
// services captured in a single profile-wide snapshot.
func BackupStamp() string { return backupStamp() }

// backupStamp is a sortable, filesystem-safe UTC timestamp used to name
// snapshots.
func backupStamp() string {
	return time.Now().UTC().Format("20060102T150405Z")
}
