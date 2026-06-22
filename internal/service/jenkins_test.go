package service

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestJenkinsValidateEnv(t *testing.T) {
	j := Jenkins{}
	if err := j.ValidateEnv(j.DefaultEnv()); err != nil {
		t.Fatalf("default env failed validation: %v", err)
	}

	env := j.DefaultEnv()
	delete(env, "host")
	if err := j.ValidateEnv(env); err == nil {
		t.Error("expected missing host to fail validation")
	}

	env = j.DefaultEnv()
	env["port"] = 70000
	if err := j.ValidateEnv(env); err == nil {
		t.Error("expected out-of-range port to fail validation")
	}

	// Optional credentials are accepted when present and well-typed.
	env = j.DefaultEnv()
	env["user"] = "admin"
	env["token"] = "11aa22bb"
	if err := j.ValidateEnv(env); err != nil {
		t.Errorf("expected credentials to validate: %v", err)
	}
}

func TestJenkinsHealth(t *testing.T) {
	ctx := context.Background()
	spec := Spec{Env: Jenkins{}.DefaultEnv()}

	// A reachable server makes Health succeed.
	ok := Jenkins{ping: func(context.Context, string) error { return nil }}
	if err := ok.Health(ctx, spec); err != nil {
		t.Errorf("Health with reachable server: %v", err)
	}

	// An unreachable server surfaces an actionable, wrapped error.
	boom := errors.New("connection refused")
	down := Jenkins{ping: func(context.Context, string) error { return boom }}
	if err := down.Health(ctx, spec); err == nil || !errors.Is(err, boom) {
		t.Errorf("Health with unreachable server: got %v, want wrapped %v", err, boom)
	}
}

func TestJenkinsHealthBaseURL(t *testing.T) {
	var got string
	j := Jenkins{ping: func(_ context.Context, baseURL string) error {
		got = baseURL
		return nil
	}}
	spec := Spec{Env: Config{"host": "ci.example", "port": 9090}}
	if err := j.Health(context.Background(), spec); err != nil {
		t.Fatalf("Health: %v", err)
	}
	if want := "http://ci.example:9090"; got != want {
		t.Errorf("base URL = %q, want %q", got, want)
	}
}

// fakeJenkins builds a Jenkins whose REST getter replies with canned bodies
// keyed by a substring of the requested path, recording the params it saw.
func fakeJenkins(t *testing.T, version string, replies map[string]string, seen *jenkinsParams) Jenkins {
	t.Helper()
	return Jenkins{get: func(_ context.Context, p jenkinsParams, path string) (jenkinsResult, error) {
		if seen != nil {
			*seen = p
		}
		for frag, body := range replies {
			if strings.Contains(path, frag) {
				return jenkinsResult{body: []byte(body), version: version}, nil
			}
		}
		t.Fatalf("unexpected jenkins path %q", path)
		return jenkinsResult{}, nil
	}}
}

func TestJenkinsInfo(t *testing.T) {
	var seen jenkinsParams
	j := fakeJenkins(t, "2.452.3", map[string]string{
		"/api/json": `{"nodeName":"","nodeDescription":"the master","mode":"NORMAL","quietingDown":false,"jobs":[{"name":"a"},{"name":"b"}]}`,
	}, &seen)
	spec := Spec{Env: Config{"host": "localhost", "port": 8080, "user": "admin", "token": "secret"}}

	info, err := j.Info(context.Background(), spec)
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if info.Version != "2.452.3" {
		t.Errorf("Version = %q, want 2.452.3", info.Version)
	}
	if info.Description != "the master" || info.Mode != "NORMAL" {
		t.Errorf("unexpected info: %+v", info)
	}
	if info.JobCount != 2 {
		t.Errorf("JobCount = %d, want 2", info.JobCount)
	}
	// Credentials from env are threaded through to the getter for Basic auth.
	if seen.user != "admin" || seen.token != "secret" {
		t.Errorf("getter saw creds %q/%q, want admin/secret", seen.user, seen.token)
	}
}

func TestJenkinsJobs(t *testing.T) {
	j := fakeJenkins(t, "", map[string]string{
		"jobs[": `{"jobs":[
			{"name":"build","url":"http://localhost:8080/job/build/","color":"blue","lastBuild":{"number":42}},
			{"name":"deploy","url":"http://localhost:8080/job/deploy/","color":"notbuilt"}
		]}`,
	}, nil)

	jobs, err := j.Jobs(context.Background(), Spec{Env: Jenkins{}.DefaultEnv()})
	if err != nil {
		t.Fatalf("Jobs: %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("got %d jobs, want 2", len(jobs))
	}
	if jobs[0].Name != "build" || jobs[0].Color != "blue" || jobs[0].LastBuild != 42 {
		t.Errorf("unexpected first job: %+v", jobs[0])
	}
	if jobs[1].LastBuild != 0 {
		t.Errorf("never-built job should have LastBuild 0, got %d", jobs[1].LastBuild)
	}
}

func TestJenkinsBuilds(t *testing.T) {
	var seen jenkinsParams
	j := fakeJenkins(t, "", map[string]string{
		"/job/": `{"builds":[
			{"number":42,"result":"SUCCESS","building":false,"timestamp":1700000000000,"duration":12000},
			{"number":43,"result":"","building":true,"timestamp":1700000100000,"duration":0}
		]}`,
	}, &seen)

	builds, err := j.Builds(context.Background(), Spec{Env: Jenkins{}.DefaultEnv()}, "my job")
	if err != nil {
		t.Fatalf("Builds: %v", err)
	}
	if len(builds) != 2 {
		t.Fatalf("got %d builds, want 2", len(builds))
	}
	if builds[0].Number != 42 || builds[0].Result != "SUCCESS" || builds[0].Duration != 12000 {
		t.Errorf("unexpected first build: %+v", builds[0])
	}
	if !builds[1].Building || builds[1].Result != "" {
		t.Errorf("running build should be building with no result: %+v", builds[1])
	}
}

func TestJenkinsBuildsEscapesJobName(t *testing.T) {
	var path string
	j := Jenkins{get: func(_ context.Context, _ jenkinsParams, p string) (jenkinsResult, error) {
		path = p
		return jenkinsResult{body: []byte(`{"builds":[]}`)}, nil
	}}
	if _, err := j.Builds(context.Background(), Spec{Env: Jenkins{}.DefaultEnv()}, "my job"); err != nil {
		t.Fatalf("Builds: %v", err)
	}
	if !strings.Contains(path, "/job/my%20job/") {
		t.Errorf("job name not path-escaped in %q", path)
	}
}
