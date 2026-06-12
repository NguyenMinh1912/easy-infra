// Object-browser endpoints: list a profile's object-store buckets and walk the
// folder-organised objects within them, for the UI's service detail page. Only
// services implementing service.Browser (minio today) support browsing.
package server

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"time"

	"github.com/minhnc/easy-infra/internal/project"
	"github.com/minhnc/easy-infra/internal/service"
)

// browseTimeout bounds one listing request so an unreachable store fails with a
// clear error instead of hanging the page.
const browseTimeout = 15 * time.Second

// bucketsResponse is the JSON shape of a bucket listing. Like the console, a
// failed listing (e.g. store unreachable) is an expected outcome: OK stays 200
// and the reason lands in Error with Buckets empty, so the UI can surface it
// without treating it as a transport error.
type bucketsResponse struct {
	Buckets []string `json:"buckets"`
	Error   string   `json:"error,omitempty"`
}

// objectsResponse is the JSON shape of one folder level within a bucket.
type objectsResponse struct {
	Prefixes []string              `json:"prefixes"`
	Objects  []service.ObjectEntry `json:"objects"`
	Error    string                `json:"error,omitempty"`
}

// handleBrowseBuckets lists the buckets of the named profile's service.
func (s *Server) handleBrowseBuckets(w http.ResponseWriter, r *http.Request) {
	browser, spec, ok := s.resolveBrowser(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), browseTimeout)
	defer cancel()
	buckets, err := browser.Buckets(ctx, spec)
	if err != nil {
		writeJSON(w, http.StatusOK, bucketsResponse{Buckets: []string{}, Error: err.Error()})
		return
	}
	if buckets == nil {
		buckets = []string{}
	}
	writeJSON(w, http.StatusOK, bucketsResponse{Buckets: buckets})
}

// handleBrowseObjects lists the immediate sub-folders and objects under the
// `prefix` query within the `bucket` query of the named profile's service.
func (s *Server) handleBrowseObjects(w http.ResponseWriter, r *http.Request) {
	browser, spec, ok := s.resolveBrowser(w, r)
	if !ok {
		return
	}
	bucket := r.URL.Query().Get("bucket")
	if bucket == "" {
		writeError(w, http.StatusBadRequest, "bucket must not be empty")
		return
	}
	prefix := r.URL.Query().Get("prefix")

	ctx, cancel := context.WithTimeout(r.Context(), browseTimeout)
	defer cancel()
	listing, err := browser.Objects(ctx, spec, bucket, prefix)
	if err != nil {
		writeJSON(w, http.StatusOK, objectsResponse{
			Prefixes: []string{}, Objects: []service.ObjectEntry{}, Error: err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, objectsResponse{Prefixes: listing.Prefixes, Objects: listing.Objects})
}

// handleBrowseObject streams one object's contents as a download. The `bucket`
// and `key` queries name the object; the response carries it as an attachment
// so the browser saves it under the object's base name.
func (s *Server) handleBrowseObject(w http.ResponseWriter, r *http.Request) {
	browser, spec, ok := s.resolveBrowser(w, r)
	if !ok {
		return
	}
	bucket := r.URL.Query().Get("bucket")
	if bucket == "" {
		writeError(w, http.StatusBadRequest, "bucket must not be empty")
		return
	}
	key := r.URL.Query().Get("key")
	if key == "" {
		writeError(w, http.StatusBadRequest, "key must not be empty")
		return
	}

	rc, info, err := browser.Object(r.Context(), spec, bucket, key)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer rc.Close()

	contentType := info.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	if info.Size > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(info.Size, 10))
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", path.Base(key)))
	// Headers are committed on the first write; a copy error mid-stream can no
	// longer change the status, so there is nothing useful to report past it.
	_, _ = io.Copy(w, rc)
}

// handleBrowseArchive streams a zip of several selected objects and/or folders.
// The `bucket` query names the store; repeated `key` queries name individual
// objects and repeated `prefix` queries name folders, whose objects are
// included recursively. Entries are stored under their full keys so the
// archive preserves the folder layout.
func (s *Server) handleBrowseArchive(w http.ResponseWriter, r *http.Request) {
	browser, spec, ok := s.resolveBrowser(w, r)
	if !ok {
		return
	}
	bucket := r.URL.Query().Get("bucket")
	if bucket == "" {
		writeError(w, http.StatusBadRequest, "bucket must not be empty")
		return
	}
	keys := r.URL.Query()["key"]
	prefixes := r.URL.Query()["prefix"]
	if len(keys) == 0 && len(prefixes) == 0 {
		writeError(w, http.StatusBadRequest, "at least one key or prefix is required")
		return
	}

	// Expand the selection to a concrete object set before writing anything: a
	// store-unreachable failure then surfaces as a 502 rather than a truncated
	// archive, since the status can no longer change once the body starts.
	listCtx, cancel := context.WithTimeout(r.Context(), browseTimeout)
	defer cancel()
	objectKeys, err := collectArchiveKeys(listCtx, browser, spec, bucket, keys, prefixes)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if len(objectKeys) == 0 {
		writeError(w, http.StatusNotFound, "selection contains no objects")
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", bucket+".zip"))

	zw := zip.NewWriter(w)
	defer zw.Close()
	for _, key := range objectKeys {
		// Headers are committed on the first write; a copy error mid-stream can no
		// longer change the status, so abandon the (now truncated) archive — that
		// truncation is the only signal left to the client.
		if err := writeArchiveEntry(r.Context(), zw, browser, spec, bucket, key); err != nil {
			return
		}
	}
}

// collectArchiveKeys resolves the selection into the de-duplicated set of object
// keys to archive: the explicit `keys` as-is, plus every object found beneath
// each folder `prefix`. Order is preserved so the archive is deterministic.
func collectArchiveKeys(ctx context.Context, browser service.Browser, spec service.Spec, bucket string, keys, prefixes []string) ([]string, error) {
	seen := make(map[string]bool)
	var out []string
	add := func(key string) {
		if key == "" || seen[key] {
			return
		}
		seen[key] = true
		out = append(out, key)
	}
	for _, key := range keys {
		add(key)
	}
	for _, prefix := range prefixes {
		if prefix == "" {
			continue
		}
		if err := walkPrefix(ctx, browser, spec, bucket, prefix, add); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// walkPrefix lists objects under prefix recursively, calling add for each key.
func walkPrefix(ctx context.Context, browser service.Browser, spec service.Spec, bucket, prefix string, add func(string)) error {
	listing, err := browser.Objects(ctx, spec, bucket, prefix)
	if err != nil {
		return err
	}
	for _, obj := range listing.Objects {
		add(obj.Key)
	}
	for _, sub := range listing.Prefixes {
		if err := walkPrefix(ctx, browser, spec, bucket, sub, add); err != nil {
			return err
		}
	}
	return nil
}

// writeArchiveEntry copies one object's contents into the zip under its key.
func writeArchiveEntry(ctx context.Context, zw *zip.Writer, browser service.Browser, spec service.Spec, bucket, key string) error {
	rc, _, err := browser.Object(ctx, spec, bucket, key)
	if err != nil {
		return err
	}
	defer rc.Close()
	dst, err := zw.Create(key)
	if err != nil {
		return err
	}
	_, err = io.Copy(dst, rc)
	return err
}

// resolveBrowser maps the {name}/{service} path onto a browse-capable service
// and the Spec for the profile's saved env. On failure it writes the error
// response and returns ok=false. It mirrors resolveQuerier.
func (s *Server) resolveBrowser(w http.ResponseWriter, r *http.Request) (service.Browser, service.Spec, bool) {
	profileName := r.PathValue("name")
	svcID := r.PathValue("service")

	proj, err := project.Load(s.paths, s.reg)
	if err != nil {
		s.writeProjectError(w, err)
		return nil, service.Spec{}, false
	}
	services, err := proj.ProfileConfig(profileName)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return nil, service.Spec{}, false
	}
	entry, ok := services[svcID]
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("service %q is not defined in profile %q", svcID, profileName))
		return nil, service.Spec{}, false
	}
	svcType := entry.ResolveType(svcID)
	svc, ok := s.reg.Get(svcType)
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("unknown service %q", svcType))
		return nil, service.Spec{}, false
	}
	browser, ok := svc.(service.Browser)
	if !ok {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("service %q does not support browsing", svcID))
		return nil, service.Spec{}, false
	}
	return browser, service.Spec{Profile: profileName, Env: entry.Config}, true
}
