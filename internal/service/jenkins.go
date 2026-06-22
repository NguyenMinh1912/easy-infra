package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Jenkins manages a Jenkins continuous-integration automation server.
//
// ping and get are seams for testing: when nil Health does a real HTTP GET
// against the login page (realJenkinsPinger) and the browser methods query the
// real REST API (realJenkinsGetter); tests set them to inject canned results.
type Jenkins struct {
	ping jenkinsPinger
	get  jenkinsGetter
	post jenkinsPoster
}

// getter returns the REST fetcher to use, defaulting to a real authed GET.
func (j Jenkins) getter() jenkinsGetter {
	if j.get != nil {
		return j.get
	}
	return realJenkinsGetter
}

// poster returns the REST mutator to use, defaulting to a real authed POST.
func (j Jenkins) poster() jenkinsPoster {
	if j.post != nil {
		return j.post
	}
	return realJenkinsPoster
}

// jenkinsPinger probes a Jenkins server's base URL and reports whether it is
// reachable and ready. It is a seam like the other services' openers: the
// zero-value Jenkins does a real HTTP GET (realJenkinsPinger), while tests
// supply a fake without a live server.
type jenkinsPinger func(ctx context.Context, baseURL string) error

// pinger returns the prober to use, defaulting to a real HTTP GET.
func (j Jenkins) pinger() jenkinsPinger {
	if j.ping != nil {
		return j.ping
	}
	return realJenkinsPinger
}

// Name implements Service.
func (Jenkins) Name() string { return "jenkins" }

// DefaultDefinition implements Service.
func (Jenkins) DefaultDefinition() Config {
	return Config{"version": "lts", cleanableKey: true}
}

// ValidateDefinition implements Service.
func (Jenkins) ValidateDefinition(cfg Config) error {
	if _, err := optionalString(cfg, "version", "lts"); err != nil {
		return err
	}
	return validateCleanable(cfg)
}

// DefaultEnv implements Service.
func (Jenkins) DefaultEnv() Config {
	return Config{
		"host": "localhost",
		"port": 8080,
	}
}

// ValidateEnv implements Service. Credentials are optional: an anonymous-read
// Jenkins needs none, while a secured one is reached with a username and an API
// token (Jenkins's recommended alternative to a password for REST access).
func (Jenkins) ValidateEnv(cfg Config) error {
	if _, err := requireString(cfg, "host"); err != nil {
		return err
	}
	if _, err := optionalPort(cfg, "port", 8080); err != nil {
		return err
	}
	if _, err := optionalString(cfg, "user", ""); err != nil {
		return err
	}
	if _, err := optionalString(cfg, "token", ""); err != nil {
		return err
	}
	return nil
}

// jenkinsBaseURL builds the base URL a Jenkins server is reached on from a
// profile's env.
func jenkinsBaseURL(env Config) (string, error) {
	host, err := requireString(env, "host")
	if err != nil {
		return "", err
	}
	port, err := optionalPort(env, "port", 8080)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("http://%s:%d", host, port), nil
}

// Health implements Service: GET the Jenkins login page and confirm the server
// answers. Jenkins serves /login regardless of whether security is enabled and
// stamps every response with an X-Jenkins header carrying its version, so a
// reachable server is a ready one.
func (j Jenkins) Health(ctx context.Context, spec Spec) error {
	baseURL, err := jenkinsBaseURL(spec.Env)
	if err != nil {
		return err
	}
	if err := j.pinger()(ctx, baseURL); err != nil {
		return fmt.Errorf("jenkins not ready: %w", err)
	}
	return nil
}

// realJenkinsPinger does a live GET against the Jenkins login page and confirms
// the response carries the X-Jenkins header that identifies a Jenkins server,
// distinguishing a real Jenkins from any other service answering on the port.
func realJenkinsPinger(ctx context.Context, baseURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/login", nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.Header.Get("X-Jenkins") == "" {
		return fmt.Errorf("%s did not identify as a Jenkins server (status %s)", baseURL, resp.Status)
	}
	return nil
}

// Info implements JenkinsBrowser: read the controller's summary from
// `/api/json` and the running version from the X-Jenkins response header.
func (j Jenkins) Info(ctx context.Context, spec Spec) (JenkinsInfo, error) {
	p, err := jenkinsParamsFrom(spec.Env)
	if err != nil {
		return JenkinsInfo{}, err
	}
	res, err := j.getter()(ctx, p, "/api/json?tree=nodeName,nodeDescription,mode,quietingDown,jobs[name]")
	if err != nil {
		return JenkinsInfo{}, fmt.Errorf("reaching jenkins: %w", err)
	}
	var raw struct {
		NodeName        string `json:"nodeName"`
		NodeDescription string `json:"nodeDescription"`
		Mode            string `json:"mode"`
		QuietingDown    bool   `json:"quietingDown"`
		Jobs            []struct {
			Name string `json:"name"`
		} `json:"jobs"`
	}
	if err := json.Unmarshal(res.body, &raw); err != nil {
		return JenkinsInfo{}, fmt.Errorf("parsing jenkins info: %w", err)
	}
	return JenkinsInfo{
		Version:      res.version,
		NodeName:     raw.NodeName,
		Description:  raw.NodeDescription,
		Mode:         raw.Mode,
		QuietingDown: raw.QuietingDown,
		JobCount:     len(raw.Jobs),
	}, nil
}

// Jobs implements JenkinsBrowser: list the controller's top-level jobs with the
// status color and last-build number Jenkins reports for each.
func (j Jenkins) Jobs(ctx context.Context, spec Spec) ([]JobInfo, error) {
	p, err := jenkinsParamsFrom(spec.Env)
	if err != nil {
		return nil, err
	}
	res, err := j.getter()(ctx, p, "/api/json?tree=jobs[name,url,color,lastBuild[number]]")
	if err != nil {
		return nil, fmt.Errorf("reaching jenkins: %w", err)
	}
	var raw struct {
		Jobs []struct {
			Name      string `json:"name"`
			URL       string `json:"url"`
			Color     string `json:"color"`
			LastBuild *struct {
				Number int64 `json:"number"`
			} `json:"lastBuild"`
		} `json:"jobs"`
	}
	if err := json.Unmarshal(res.body, &raw); err != nil {
		return nil, fmt.Errorf("parsing jenkins jobs: %w", err)
	}
	jobs := make([]JobInfo, 0, len(raw.Jobs))
	for _, jb := range raw.Jobs {
		info := JobInfo{Name: jb.Name, URL: jb.URL, Color: jb.Color}
		if jb.LastBuild != nil {
			info.LastBuild = jb.LastBuild.Number
		}
		jobs = append(jobs, info)
	}
	return jobs, nil
}

// Builds implements JenkinsBrowser: list the named job's recent builds, most
// recent first (the order Jenkins returns them).
func (j Jenkins) Builds(ctx context.Context, spec Spec, job string) ([]BuildInfo, error) {
	p, err := jenkinsParamsFrom(spec.Env)
	if err != nil {
		return nil, err
	}
	path := "/job/" + url.PathEscape(job) + "/api/json?tree=builds[number,result,building,timestamp,duration]"
	res, err := j.getter()(ctx, p, path)
	if err != nil {
		return nil, fmt.Errorf("reaching jenkins: %w", err)
	}
	var raw struct {
		Builds []BuildInfo `json:"builds"`
	}
	if err := json.Unmarshal(res.body, &raw); err != nil {
		return nil, fmt.Errorf("parsing jenkins builds: %w", err)
	}
	if raw.Builds == nil {
		raw.Builds = []BuildInfo{}
	}
	return raw.Builds, nil
}

// BuildLog implements JenkinsBrowser: fetch a chunk of a build's console output
// from Jenkins's progressive-text endpoint, starting at byte offset start. The
// response carries the new output in the body, the total size so far in the
// X-Text-Size header (the next offset), and whether the build is still running
// in X-More-Data — the three pieces a long-polling viewer needs.
func (j Jenkins) BuildLog(ctx context.Context, spec Spec, job string, number, start int64) (LogChunk, error) {
	p, err := jenkinsParamsFrom(spec.Env)
	if err != nil {
		return LogChunk{}, err
	}
	path := fmt.Sprintf("/job/%s/%d/logText/progressiveText?start=%d", url.PathEscape(job), number, start)
	res, err := j.getter()(ctx, p, path)
	if err != nil {
		return LogChunk{}, fmt.Errorf("reaching jenkins: %w", err)
	}
	// Default the next offset to start plus what we read, in case the server
	// omits X-Text-Size; prefer the header when present as it is authoritative.
	chunk := LogChunk{Text: string(res.body), Offset: start + int64(len(res.body))}
	if res.header != nil {
		if size, err := strconv.ParseInt(res.header.Get("X-Text-Size"), 10, 64); err == nil {
			chunk.Offset = size
		}
		chunk.More = strings.EqualFold(res.header.Get("X-More-Data"), "true")
	}
	return chunk, nil
}

// TriggerBuild implements JenkinsBrowser: POST to the named job's build endpoint
// to schedule a new (parameterless) build.
func (j Jenkins) TriggerBuild(ctx context.Context, spec Spec, job string) error {
	p, err := jenkinsParamsFrom(spec.Env)
	if err != nil {
		return err
	}
	path := "/job/" + url.PathEscape(job) + "/build"
	if err := j.poster()(ctx, p, path); err != nil {
		return fmt.Errorf("triggering build of %q: %w", job, err)
	}
	return nil
}

// Lifecycle provisioning (Apply/Backup/Clean) is the per-service seam for
// Docker-backed provisioning, which is future work; until a provider lands they
// report ErrNotImplemented.

// Apply implements Service.
func (Jenkins) Apply(context.Context, Spec) error { return notImplemented("jenkins", "apply") }

// Backup implements Service.
func (Jenkins) Backup(context.Context, Spec) error { return notImplemented("jenkins", "backup") }

// Clean implements Service.
func (Jenkins) Clean(_ context.Context, spec Spec) error {
	if err := spec.ensureCleanable(); err != nil {
		return err
	}
	return notImplemented("jenkins", "clean")
}
