package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
)

// jenkinsParams is a profile's Jenkins connection settings, normalised out of
// the discrete env fields so the client treats them uniformly. Credentials are
// optional: an anonymous-read server leaves user/token empty, while a secured
// one is reached with a username and an API token (or password) over Basic auth.
type jenkinsParams struct {
	host  string
	port  int
	user  string
	token string
}

// baseURL is the root URL the Jenkins server is reached on.
func (p jenkinsParams) baseURL() string {
	return fmt.Sprintf("http://%s:%d", p.host, p.port)
}

// jenkinsParamsFrom extracts the connection settings from a profile's env.
func jenkinsParamsFrom(env Config) (jenkinsParams, error) {
	host, err := requireString(env, "host")
	if err != nil {
		return jenkinsParams{}, err
	}
	port, err := optionalPort(env, "port", 8080)
	if err != nil {
		return jenkinsParams{}, err
	}
	user, err := optionalString(env, "user", "")
	if err != nil {
		return jenkinsParams{}, err
	}
	token, err := optionalString(env, "token", "")
	if err != nil {
		return jenkinsParams{}, err
	}
	return jenkinsParams{host: host, port: port, user: user, token: token}, nil
}

// jenkinsResult is one Jenkins REST response: the JSON body, the running server
// version read from the X-Jenkins response header (the only place Jenkins
// reports it), and the full response header for callers that need others (the
// progressive log reads X-Text-Size / X-More-Data from it).
type jenkinsResult struct {
	body    []byte
	version string
	header  http.Header
}

// jenkinsGetter performs an authenticated GET against a Jenkins REST path
// (e.g. "/api/json?tree=…") and returns the body and version header. It is a
// seam like the other services' openers: the zero-value Jenkins does a real
// HTTP GET (realJenkinsGetter), while tests supply canned results without a
// live server.
type jenkinsGetter func(ctx context.Context, p jenkinsParams, path string) (jenkinsResult, error)

// realJenkinsGetter does the live GET against the Jenkins server, attaching
// Basic auth when credentials are configured and capping the body so a
// misbehaving server can't stream unbounded data.
func realJenkinsGetter(ctx context.Context, p jenkinsParams, path string) (jenkinsResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL()+path, nil)
	if err != nil {
		return jenkinsResult{}, err
	}
	if p.user != "" {
		req.SetBasicAuth(p.user, p.token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return jenkinsResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return jenkinsResult{}, fmt.Errorf("%s returned %s", path, resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return jenkinsResult{}, err
	}
	return jenkinsResult{body: body, version: resp.Header.Get("X-Jenkins"), header: resp.Header}, nil
}

// jenkinsPoster performs an authenticated POST against a Jenkins path (e.g.
// "/job/x/build"), handling the CSRF crumb. It is a seam like jenkinsGetter:
// the zero-value Jenkins POSTs against a real server (realJenkinsPoster), while
// tests supply a fake without a live server.
type jenkinsPoster func(ctx context.Context, p jenkinsParams, path string) error

// realJenkinsPoster fetches a CSRF crumb (when the server requires one) and
// POSTs to the path with Basic auth. Jenkins answers a successful trigger with
// 201 Created (the queued item's Location), so 2xx and 3xx are all accepted.
//
// The crumb and the POST share one client with a cookie jar: Jenkins binds the
// crumb to the session it sets via a JSESSIONID cookie on the crumb response,
// so the POST must carry that cookie back or the crumb is rejected with 403.
func realJenkinsPoster(ctx context.Context, p jenkinsParams, path string) error {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return err
	}
	client := &http.Client{Jar: jar}

	crumbField, crumbValue, err := jenkinsCrumb(ctx, client, p)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL()+path, nil)
	if err != nil {
		return err
	}
	if p.user != "" {
		req.SetBasicAuth(p.user, p.token)
	}
	if crumbField != "" {
		req.Header.Set(crumbField, crumbValue)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s returned %s%s", path, resp.Status, jenkinsErrorDetail(resp))
	}
	return nil
}

// jenkinsErrorDetail extracts a short, human-readable reason from a failed
// Jenkins response so a bare "403 Forbidden" carries why it failed (e.g. a
// missing crumb or a missing Build permission). Jenkins reports the reason in
// the X-Error header, or in the body as plain text; HTML error pages are
// skipped since they carry no concise message.
func jenkinsErrorDetail(resp *http.Response) string {
	if msg := strings.TrimSpace(resp.Header.Get("X-Error")); msg != "" {
		return ": " + msg
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<12))
	msg := strings.TrimSpace(string(body))
	if msg == "" || strings.HasPrefix(msg, "<") {
		return ""
	}
	if len(msg) > 200 {
		msg = msg[:200]
	}
	return ": " + msg
}

// jenkinsCrumb fetches a CSRF crumb from the server's crumb issuer using the
// given client, returning the header field name and value to attach to a
// mutating request. The client's cookie jar captures the session cookie the
// crumb is bound to. CSRF protection can be disabled, in which case the issuer
// answers 404; that is not an error — empty values are returned and the caller
// sends no crumb.
func jenkinsCrumb(ctx context.Context, client *http.Client, p jenkinsParams) (field, value string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL()+"/crumbIssuer/api/json", nil)
	if err != nil {
		return "", "", err
	}
	if p.user != "" {
		req.SetBasicAuth(p.user, p.token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return "", "", nil // CSRF protection disabled.
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("fetching crumb: %s", resp.Status)
	}
	var crumb struct {
		Crumb             string `json:"crumb"`
		CrumbRequestField string `json:"crumbRequestField"`
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if err != nil {
		return "", "", err
	}
	if err := json.Unmarshal(body, &crumb); err != nil {
		return "", "", fmt.Errorf("parsing crumb: %w", err)
	}
	return crumb.CrumbRequestField, crumb.Crumb, nil
}
