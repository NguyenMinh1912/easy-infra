package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
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

// jenkinsResult is one Jenkins REST response: the JSON body plus the running
// server version read from the X-Jenkins response header (the only place
// Jenkins reports it).
type jenkinsResult struct {
	body    []byte
	version string
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
	return jenkinsResult{body: body, version: resp.Header.Get("X-Jenkins")}, nil
}
