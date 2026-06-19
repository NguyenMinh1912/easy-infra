// Cloud-browser endpoints: list a profile's LocalStack SQS queues and SES
// identities, for the UI's LocalStack detail page. Only services implementing
// service.CloudBrowser (localstack today) support browsing cloud resources.
package server

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"strconv"
	"strings"
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

// messagesResponse is the JSON shape of an SQS queue's message preview. Like the
// queue listing, an unreachable endpoint stays 200 with the reason in Error and
// Messages empty.
type messagesResponse struct {
	Messages []service.MessageInfo `json:"messages"`
	Error    string                `json:"error,omitempty"`
}

// identitiesResponse is the JSON shape of the SES identity listing.
type identitiesResponse struct {
	Identities []service.IdentityInfo `json:"identities"`
	Error      string                 `json:"error,omitempty"`
}

// messagesResponse is the JSON shape of an identity's SES message listing. Like
// the other listings, an unreachable endpoint stays 200 with the reason in
// Error and Messages empty.
type messagesResponse struct {
	Messages []service.MessageInfo `json:"messages"`
	Error    string                `json:"error,omitempty"`
}

// healthResponse is the JSON shape of the LocalStack health snapshot. Like the
// listings, an unreachable endpoint stays 200 with the reason in Error and the
// service map empty, so the UI can render an "unreachable" state.
type healthResponse struct {
	Version  string            `json:"version,omitempty"`
	Edition  string            `json:"edition,omitempty"`
	Services map[string]string `json:"services"`
	Error    string            `json:"error,omitempty"`
}

// handleCloudHealth reports the named profile's LocalStack health snapshot —
// the per-service state map and reported version — driving the overview's
// service cards and Configuration panel.
func (s *Server) handleCloudHealth(w http.ResponseWriter, r *http.Request) {
	browser, spec, ok := s.resolveCloudBrowser(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), cloudTimeout)
	defer cancel()
	health, err := browser.CloudHealth(ctx, spec)
	if err != nil {
		writeJSON(w, http.StatusOK, healthResponse{Services: map[string]string{}, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, healthResponse{
		Version:  health.Version,
		Edition:  health.Edition,
		Services: health.Services,
	})
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

// createQueueRequest is the JSON body of a create-queue request.
type createQueueRequest struct {
	Name string `json:"name"`
}

// handleCloudCreateQueue creates a new SQS queue in the named profile.
func (s *Server) handleCloudCreateQueue(w http.ResponseWriter, r *http.Request) {
	browser, spec, ok := s.resolveCloudBrowser(w, r)
	if !ok {
		return
	}
	var req createQueueRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "queue name must not be empty")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), cloudTimeout)
	defer cancel()
	if err := browser.CreateQueue(ctx, spec, name); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleCloudDeleteQueue deletes the queue identified by the `url` query
// parameter — the queue URL the listing already carries.
func (s *Server) handleCloudDeleteQueue(w http.ResponseWriter, r *http.Request) {
	browser, spec, ok := s.resolveCloudBrowser(w, r)
	if !ok {
		return
	}
	url := r.URL.Query().Get("url")
	if url == "" {
		writeError(w, http.StatusBadRequest, "queue url must not be empty")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), cloudTimeout)
	defer cancel()
	if err := browser.DeleteQueue(ctx, spec, url); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleCloudPurgeQueue removes all messages from the queue identified by the
// `url` query parameter, leaving the queue in place.
func (s *Server) handleCloudPurgeQueue(w http.ResponseWriter, r *http.Request) {
	browser, spec, ok := s.resolveCloudBrowser(w, r)
	if !ok {
		return
	}
	url := r.URL.Query().Get("url")
	if url == "" {
		writeError(w, http.StatusBadRequest, "queue url must not be empty")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), cloudTimeout)
	defer cancel()
	if err := browser.PurgeQueue(ctx, spec, url); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleCloudQueueMessages previews the messages on the queue identified by the
// `url` query parameter — a non-destructive peek for the queue's message list
// view. An optional `limit` caps how many to return (SQS allows up to 10).
func (s *Server) handleCloudQueueMessages(w http.ResponseWriter, r *http.Request) {
	browser, spec, ok := s.resolveCloudBrowser(w, r)
	if !ok {
		return
	}
	url := r.URL.Query().Get("url")
	if url == "" {
		writeError(w, http.StatusBadRequest, "queue url must not be empty")
		return
	}
	limit := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "limit must be a number")
			return
		}
		limit = n
	}

	ctx, cancel := context.WithTimeout(r.Context(), cloudTimeout)
	defer cancel()
	messages, err := browser.Messages(ctx, spec, url, limit)
	if err != nil {
		writeJSON(w, http.StatusOK, messagesResponse{Messages: []service.MessageInfo{}, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, messagesResponse{Messages: messages})
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

// handleCloudMessages lists the SES messages involving the identity named by
// the `identity` query parameter, for the identity's mail list page.
func (s *Server) handleCloudMessages(w http.ResponseWriter, r *http.Request) {
	browser, spec, ok := s.resolveCloudBrowser(w, r)
	if !ok {
		return
	}
	identity := r.URL.Query().Get("identity")
	if identity == "" {
		writeError(w, http.StatusBadRequest, "identity must not be empty")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), cloudTimeout)
	defer cancel()
	messages, err := browser.Messages(ctx, spec, identity)
	if err != nil {
		writeJSON(w, http.StatusOK, messagesResponse{Messages: []service.MessageInfo{}, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, messagesResponse{Messages: messages})
}

// createIdentityRequest is the JSON body of a create-identity request.
type createIdentityRequest struct {
	Identity string `json:"identity"`
}

// handleCloudCreateIdentity registers a new SES identity for verification in the
// named profile. An identity containing "@" is verified as an email address,
// otherwise as a domain.
func (s *Server) handleCloudCreateIdentity(w http.ResponseWriter, r *http.Request) {
	browser, spec, ok := s.resolveCloudBrowser(w, r)
	if !ok {
		return
	}
	var req createIdentityRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	identity := strings.TrimSpace(req.Identity)
	if identity == "" {
		writeError(w, http.StatusBadRequest, "identity must not be empty")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), cloudTimeout)
	defer cancel()
	if err := browser.CreateIdentity(ctx, spec, identity); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleCloudDeleteIdentity removes the SES identity named by the `identity`
// query parameter — the identity name the listing already carries.
func (s *Server) handleCloudDeleteIdentity(w http.ResponseWriter, r *http.Request) {
	browser, spec, ok := s.resolveCloudBrowser(w, r)
	if !ok {
		return
	}
	identity := r.URL.Query().Get("identity")
	if identity == "" {
		writeError(w, http.StatusBadRequest, "identity must not be empty")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), cloudTimeout)
	defer cancel()
	if err := browser.DeleteIdentity(ctx, spec, identity); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

	// The overview's region dropdown is the single source of truth for which
	// region resources are queried in. When it passes ?region=, override the
	// profile's saved region for this request without mutating the stored env.
	env := entry.Config
	if region := r.URL.Query().Get("region"); region != "" {
		env = withRegion(env, region)
	}
	return browser, service.Spec{Profile: profileName, Env: env}, true
}

// withRegion returns a copy of env with its region set to the given value,
// leaving the stored config untouched.
func withRegion(env service.Config, region string) service.Config {
	clone := make(service.Config, len(env)+1)
	maps.Copy(clone, env)
	clone["region"] = region
	return clone
}
