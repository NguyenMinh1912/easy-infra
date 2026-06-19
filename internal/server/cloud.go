// Cloud-browser endpoints: list a profile's LocalStack SQS queues and SES
// identities, for the UI's LocalStack detail page. Only services implementing
// service.CloudBrowser (localstack today) support browsing cloud resources.
package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/minhnc/easy-infra/internal/service"
)

// cloudTimeout bounds one cloud request so an unreachable endpoint fails with a
// clear error instead of hanging the page. It matches keyspaceTimeout.
const cloudTimeout = 15 * time.Second

// queuesResponse is the JSON shape of the SQS queue listing. Like the keyspace
// endpoints, an unreachable endpoint is an expected outcome: OK stays 200 and
// the reason lands in Error with Queues empty.
type queuesResponse struct {
	Queues []service.QueueInfo `json:"queues"`
	Error  string              `json:"error,omitempty"`
}

// identitiesResponse is the JSON shape of the SES identity listing.
type identitiesResponse struct {
	Identities []service.IdentityInfo `json:"identities"`
	Error      string                 `json:"error,omitempty"`
}

// handleCloudQueues lists the named profile's SQS queues with their message
// counts, for the SQS detail page.
func (s *Server) handleCloudQueues(w http.ResponseWriter, r *http.Request) {
	browser, spec, ok := s.resolveCloudBrowser(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), cloudTimeout)
	defer cancel()
	queues, err := browser.Queues(ctx, spec)
	if err != nil {
		writeJSON(w, http.StatusOK, queuesResponse{Queues: []service.QueueInfo{}, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, queuesResponse{Queues: queues})
}

// handleCloudIdentities lists the named profile's SES identities with their
// verification status, for the SES detail page.
func (s *Server) handleCloudIdentities(w http.ResponseWriter, r *http.Request) {
	browser, spec, ok := s.resolveCloudBrowser(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), cloudTimeout)
	defer cancel()
	identities, err := browser.Identities(ctx, spec)
	if err != nil {
		writeJSON(w, http.StatusOK, identitiesResponse{Identities: []service.IdentityInfo{}, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, identitiesResponse{Identities: identities})
}

// resolveCloudBrowser maps the {name}/{service} path onto a cloud-browse-capable
// service and the Spec for the profile's saved env. On failure it writes the
// error response and returns ok=false. It mirrors resolveKeyBrowser.
func (s *Server) resolveCloudBrowser(w http.ResponseWriter, r *http.Request) (service.CloudBrowser, service.Spec, bool) {
	profileName := r.PathValue("name")
	svcID := r.PathValue("service")

	proj, err := s.activeProject()
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
	browser, ok := svc.(service.CloudBrowser)
	if !ok {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("service %q does not support a cloud browser", svcID))
		return nil, service.Spec{}, false
	}
	return browser, service.Spec{Profile: profileName, Env: entry.Config}, true
}
