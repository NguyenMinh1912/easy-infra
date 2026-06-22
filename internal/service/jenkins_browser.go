package service

import "context"

// JenkinsBrowser is an optional capability a Service implements when it exposes
// a Jenkins server that can be inspected — the backend of the UI's Jenkins
// detail page. Callers type-assert for it and degrade gracefully when a service
// does not provide it, mirroring how CloudBrowser models the LocalStack detail
// page and KeyBrowser the Redis keyspace. Jenkins is the only implementer today.
//
// The listing methods query the Jenkins REST API (`/api/json`) and shape the
// result for JSON; TriggerBuild is the one mutation, fetching a CSRF crumb and
// POSTing to the job's build endpoint — bundling reads and writes in one
// capability the way CloudBrowser groups its queue listing with create/delete.
type JenkinsBrowser interface {
	// Info reports the server's identity: its running version (from the
	// X-Jenkins response header), node description, mode and job count — the
	// backend of the overview's instance card.
	Info(ctx context.Context, spec Spec) (JenkinsInfo, error)

	// Jobs lists the server's top-level jobs with their last-build status — the
	// backend of the jobs table.
	Jobs(ctx context.Context, spec Spec) ([]JobInfo, error)

	// Builds lists the recent builds of the named job, most recent first — the
	// backend of a job's build history.
	Builds(ctx context.Context, spec Spec, job string) ([]BuildInfo, error)

	// BuildLog returns a chunk of the named job's build console output starting
	// at byte offset start — the backend of the build-log dialog's incremental
	// (long-polling) updates. The chunk carries the next offset to request and
	// whether the build may still produce more output, so a viewer can append
	// new lines and keep polling until the build finishes.
	BuildLog(ctx context.Context, spec Spec, job string, number, start int64) (LogChunk, error)

	// TriggerBuild schedules a new build of the named job. It is parameterless:
	// the job runs with its default parameters, if any. The build is enqueued
	// asynchronously, so this returns once Jenkins accepts the request, not when
	// the build finishes.
	TriggerBuild(ctx context.Context, spec Spec, job string) error
}

// JenkinsInfo is a Jenkins server's identity and summary state, shaped for JSON.
type JenkinsInfo struct {
	// Version is the running Jenkins version (e.g. "2.452.3"), read from the
	// X-Jenkins response header, when reported.
	Version string `json:"version,omitempty"`
	// NodeName is the controller node's name; empty for the built-in node.
	NodeName string `json:"nodeName,omitempty"`
	// Description is the controller node's description, when set.
	Description string `json:"description,omitempty"`
	// Mode is the controller's usage mode ("NORMAL" or "EXCLUSIVE").
	Mode string `json:"mode,omitempty"`
	// QuietingDown is true when the server is preparing to shut down.
	QuietingDown bool `json:"quietingDown"`
	// JobCount is the number of top-level jobs on the server.
	JobCount int `json:"jobCount"`
}

// JobInfo is one Jenkins job and its last-build status, shaped for JSON.
type JobInfo struct {
	// Name is the job name.
	Name string `json:"name"`
	// URL is the job's fully-qualified URL.
	URL string `json:"url"`
	// Color is Jenkins's raw status color for the job ("blue", "red", "yellow",
	// "disabled", "notbuilt", or a "…_anime" variant while a build runs). The UI
	// maps it to a label and badge, mirroring how the LocalStack page maps a
	// service's raw health state.
	Color string `json:"color"`
	// LastBuild is the most recent build number, 0 when the job has never built.
	LastBuild int64 `json:"lastBuild,omitempty"`
}

// LogChunk is a slice of a build's console output, shaped for JSON. It is the
// unit of the progressive log the dialog polls: Text is the new output since the
// requested offset, Offset is where the next poll should resume, and More
// reports whether the build may still append output.
type LogChunk struct {
	// Text is the console output appended since the requested start offset.
	Text string `json:"text"`
	// Offset is the byte offset to pass as start on the next poll.
	Offset int64 `json:"offset"`
	// More is true while the build is still producing output.
	More bool `json:"more"`
}

// BuildInfo is one build of a Jenkins job, shaped for JSON.
type BuildInfo struct {
	// Number is the build number.
	Number int64 `json:"number"`
	// Result is the build outcome ("SUCCESS", "FAILURE", "UNSTABLE", "ABORTED"),
	// empty while the build is still running.
	Result string `json:"result,omitempty"`
	// Building is true while the build is in progress.
	Building bool `json:"building"`
	// Timestamp is when the build started, in Unix milliseconds.
	Timestamp int64 `json:"timestamp"`
	// Duration is the build's duration in milliseconds, 0 while still running.
	Duration int64 `json:"duration"`
}
