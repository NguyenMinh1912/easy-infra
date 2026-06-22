// Jenkins detail endpoints: a profile's Jenkins instance info, its jobs, and a
// job's recent builds — the backend of the UI's Jenkins detail page. Only
// services implementing service.JenkinsBrowser (jenkins today) support these.
// As with the cloud and keyspace endpoints an unreachable server is an expected
// outcome: the status stays 200 and the reason lands in the response's Error
// field, so the UI can render an "unreachable" state.
package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/minhnc/easy-infra/internal/service"
)

// jenkinsTimeout bounds one Jenkins request so an unreachable server fails with
// a clear error instead of hanging the page. It matches cloudTimeout.
const jenkinsTimeout = 15 * time.Second

// jenkinsInfoResponse is the JSON shape of the instance-info endpoint.
type jenkinsInfoResponse struct {
	service.JenkinsInfo
	Error string `json:"error,omitempty"`
}

// jenkinsJobsResponse is the JSON shape of the jobs listing.
type jenkinsJobsResponse struct {
	Jobs  []service.JobInfo `json:"jobs"`
	Error string            `json:"error,omitempty"`
}

// jenkinsBuildsResponse is the JSON shape of a job's build history.
type jenkinsBuildsResponse struct {
	Builds []service.BuildInfo `json:"builds"`
	Error  string              `json:"error,omitempty"`
}

// handleJenkinsInfo reports the named profile's Jenkins instance info — version,
// node and job count — driving the overview's instance card.
func (s *Server) handleJenkinsInfo(w http.ResponseWriter, r *http.Request) {
	browser, spec, ok := s.resolveJenkinsBrowser(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), jenkinsTimeout)
	defer cancel()
	info, err := browser.Info(ctx, spec)
	if err != nil {
		writeJSON(w, http.StatusOK, jenkinsInfoResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, jenkinsInfoResponse{JenkinsInfo: info})
}

// handleJenkinsJobs lists the named profile's Jenkins jobs with their last-build
// status, for the jobs table.
func (s *Server) handleJenkinsJobs(w http.ResponseWriter, r *http.Request) {
	browser, spec, ok := s.resolveJenkinsBrowser(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), jenkinsTimeout)
	defer cancel()
	jobs, err := browser.Jobs(ctx, spec)
	if err != nil {
		writeJSON(w, http.StatusOK, jenkinsJobsResponse{Jobs: []service.JobInfo{}, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, jenkinsJobsResponse{Jobs: jobs})
}

// handleJenkinsBuilds lists the recent builds of the job named by the `job`
// query parameter, for a job's build history.
func (s *Server) handleJenkinsBuilds(w http.ResponseWriter, r *http.Request) {
	browser, spec, ok := s.resolveJenkinsBrowser(w, r)
	if !ok {
		return
	}
	job := strings.TrimSpace(r.URL.Query().Get("job"))
	if job == "" {
		writeError(w, http.StatusBadRequest, "job name must not be empty")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), jenkinsTimeout)
	defer cancel()
	builds, err := browser.Builds(ctx, spec, job)
	if err != nil {
		writeJSON(w, http.StatusOK, jenkinsBuildsResponse{Builds: []service.BuildInfo{}, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, jenkinsBuildsResponse{Builds: builds})
}

// resolveJenkinsBrowser maps the {name}/{service} path onto a Jenkins-browse-
// capable service and the Spec for the profile's saved env. On failure it
// writes the error response and returns ok=false. It mirrors resolveCloudBrowser.
func (s *Server) resolveJenkinsBrowser(w http.ResponseWriter, r *http.Request) (service.JenkinsBrowser, service.Spec, bool) {
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
	browser, ok := svc.(service.JenkinsBrowser)
	if !ok {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("service %q does not support a jenkins browser", svcID))
		return nil, service.Spec{}, false
	}
	return browser, service.Spec{Profile: profileName, Env: entry.Config}, true
}
