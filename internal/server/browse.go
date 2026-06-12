// Object-browser endpoints: list a profile's object-store buckets and walk the
// folder-organised objects within them, for the UI's service detail page. Only
// services implementing service.Browser (minio today) support browsing.
package server

import (
	"context"
	"fmt"
	"net/http"
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

// resolveBrowser maps the {name}/{service} path onto a browse-capable service
// and the Spec for the profile's saved env. On failure it writes the error
// response and returns ok=false. It mirrors resolveQuerier.
func (s *Server) resolveBrowser(w http.ResponseWriter, r *http.Request) (service.Browser, service.Spec, bool) {
	profileName := r.PathValue("name")
	svcName := r.PathValue("service")

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
	env, ok := services[svcName]
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("service %q is not defined in profile %q", svcName, profileName))
		return nil, service.Spec{}, false
	}
	svc, ok := s.reg.Get(svcName)
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("unknown service %q", svcName))
		return nil, service.Spec{}, false
	}
	browser, ok := svc.(service.Browser)
	if !ok {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("service %q does not support browsing", svcName))
		return nil, service.Spec{}, false
	}
	return browser, service.Spec{Profile: profileName, Env: env}, true
}
