package cli_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tcli "github.com/nicerobot/tools.admin/internal/cli"
	"github.com/nicerobot/tools.admin/internal/gitsvc"
)

type fakeDoer struct {
	fn func(*http.Request) (*http.Response, error)
}

func (f fakeDoer) Do(r *http.Request) (*http.Response, error) { return f.fn(r) }

func httpResp(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}
}

func newEnv(out io.Writer, token string, doerFn func(*http.Request) (*http.Response, error), gitRun gitsvc.RunFunc, env map[string]string) tcli.Env {
	return tcli.Env{
		Out:      out,
		Doer:     fakeDoer{doerFn},
		BaseURL:  "https://api.test",
		Getenv:   func(k string) string { return env[k] },
		Now:      func() time.Time { return time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC) },
		GitRun:   gitRun,
		ReadFile: func(string) ([]byte, error) { return []byte(""), nil },
		Mkdir:    func(string) error { return nil },
		WriteOut: func(string, []byte) error { return nil },
		Glob:     func(string) ([]string, error) { return nil, nil },
		Remove:   func(string) error { return nil },
	}
}

func run(env tcli.Env, args ...string) error {
	return tcli.NewCommand(env).Run(context.Background(), append([]string{"tools.admin"}, args...))
}

func TestSnapshotActionSuccess(t *testing.T) {
	var out strings.Builder
	doer := func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/users/myorg":
			return httpResp(200, `{"login":"myorg","type":"Organization"}`), nil
		case "/orgs/myorg/repos":
			return httpResp(200, `[{"name":"a"}]`), nil
		}
		return httpResp(404, `{}`), nil
	}
	env := newEnv(&out, "tok", doer, nil, map[string]string{"GH_TOKEN": "tok"})
	require.NoError(t, run(env, "snapshot", "--owner", "myorg"))
	assert.Contains(t, out.String(), "  wrote .github/repos/a.yml")
}

func TestSnapshotActionNoToken(t *testing.T) {
	var out strings.Builder
	env := newEnv(&out, "", func(*http.Request) (*http.Response, error) { return httpResp(200, `{}`), nil }, nil, nil)
	require.Error(t, run(env, "snapshot", "--owner", "myorg"))
}

func TestCreatePRActionNoChanges(t *testing.T) {
	var out strings.Builder
	gitRun := func([]string) (gitsvc.Result, error) { return gitsvc.Result{ExitCode: 0}, nil }
	env := newEnv(&out, "tok", nil, gitRun, nil)
	require.NoError(t, run(env, "create-pr"))
	assert.Contains(t, out.String(), "No changes to commit.")
}

func TestCleanupRunsActionSuccess(t *testing.T) {
	var out strings.Builder
	doer := func(*http.Request) (*http.Response, error) {
		return httpResp(200, `{"workflow_runs":[]}`), nil
	}
	env := newEnv(&out, "tok", doer, nil, map[string]string{"GH_TOKEN": "tok"})
	require.NoError(t, run(env, "cleanup-runs", "--owner", "nicerobot", "--repo", "repo1", "--days", "7", "--keep", "3", "--dry-run"))
	assert.Contains(t, out.String(), "Summary: 1 repos scanned")
}

func TestCleanupRunsActionNoToken(t *testing.T) {
	var out strings.Builder
	env := newEnv(&out, "", func(*http.Request) (*http.Response, error) { return httpResp(200, `{}`), nil }, nil, nil)
	require.Error(t, run(env, "cleanup-runs", "--owner", "x"))
}
