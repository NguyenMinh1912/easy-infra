package service

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Backups are organised as versioned snapshots. Each snapshot is a single
// timestamped folder under a profile, and every service in that snapshot writes
// its artifact into the same folder:
//
//	.easy-infra/backups/<profile>/<snapshot>/<service>...
//
// so one snapshot captures the whole profile at a point in time. Snapshot ids
// are sortable UTC timestamps, so the lexically greatest is the newest.

// BackupsDir is the directory holding a profile's backup snapshots, relative to
// the project root (matching how config/state paths are resolved).
func BackupsDir(profile string) string {
	return filepath.Join(".easy-infra", "backups", profile)
}

// NewSnapshotDir returns the path for a fresh snapshot of profile, named by a
// sortable UTC timestamp. The command layer creates this once and shares it
// across every service so they all land in the same folder.
func NewSnapshotDir(profile string) string {
	return filepath.Join(BackupsDir(profile), backupStamp())
}

// ListSnapshots returns a profile's snapshot ids (the timestamp folder names) in
// sorted order — oldest first, newest last. A missing backups dir yields an
// empty list rather than an error.
func ListSnapshots(profile string) ([]string, error) {
	entries, err := os.ReadDir(BackupsDir(profile))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading backups %s: %w", BackupsDir(profile), err)
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

// SnapshotDir returns the path of the snapshot folder named id under profile.
// The id is a folder name produced by NewSnapshotDir and listed by
// ListSnapshots; callers validate it exists before trusting client input.
func SnapshotDir(profile, id string) string {
	return filepath.Join(BackupsDir(profile), id)
}

// latestSnapshotDir returns the path of the newest snapshot folder for profile,
// or an empty string when none exist.
func latestSnapshotDir(profile string) (string, error) {
	ids, err := ListSnapshots(profile)
	if err != nil || len(ids) == 0 {
		return "", err
	}
	return filepath.Join(BackupsDir(profile), ids[len(ids)-1]), nil
}

// backupStamp is a sortable, filesystem-safe UTC timestamp used to name
// snapshots.
func backupStamp() string {
	return time.Now().UTC().Format("20060102T150405Z")
}
