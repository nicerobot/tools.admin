package cleanupruns

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicerobot/tools.admin/internal/constants"
	"github.com/nicerobot/tools.admin/internal/github"
	"github.com/nicerobot/tools.admin/internal/repo"
)

type deleteCall struct {
	owner repo.Owner
	name  repo.Name
	id    repo.RunID
}

type fakeGH struct {
	listReposErr    error
	listRunsErr     error
	deleteErr       error
	runs            map[string][]github.WorkflowRun
	lastBefore      repo.CreatedBefore
	repos           []github.Repository
	deletes         []deleteCall
	listReposCalled bool
}

func (f *fakeGH) ListRepos(repo.Owner) ([]github.Repository, error) {
	f.listReposCalled = true
	return f.repos, f.listReposErr
}

func (f *fakeGH) ListWorkflowRuns(_ repo.Owner, r repo.Name, b repo.CreatedBefore) ([]github.WorkflowRun, error) {
	f.lastBefore = b
	if f.listRunsErr != nil {
		return nil, f.listRunsErr
	}
	return f.runs[string(r)], nil
}

func (f *fakeGH) DeleteWorkflowRun(o repo.Owner, r repo.Name, id repo.RunID) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deletes = append(f.deletes, deleteCall{o, r, id})
	return nil
}

func mkRun(id, wf int64, created string) github.WorkflowRun {
	return github.WorkflowRun{ID: id, WorkflowID: wf, CreatedAt: created, Name: "CI", Status: "completed"}
}

func mkRepo(name string) github.Repository { return github.Repository{Name: name} }

func fixedNow() time.Time { return time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC) }

func runCleanup(t *testing.T, gh *fakeGH, env map[string]string, cfg Config) (Result, error) {
	t.Helper()
	orig := deps
	t.Cleanup(func() { deps = orig })
	deps = func() (dependencies, error) {
		return dependencies{
			github: gh,
			getenv: func(k string) string { return env[k] },
			now:    fixedNow,
		}, nil
	}
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	return Run(context.Background(), logger, cfg)
}

func TestDeletesOldRuns(t *testing.T) {
	gh := &fakeGH{runs: map[string][]github.WorkflowRun{
		"repo1": {
			mkRun(1, 1, "2025-01-01T00:00:00Z"),
			mkRun(2, 1, "2025-01-02T00:00:00Z"),
			mkRun(3, 1, "2025-01-03T00:00:00Z"),
		},
	}}
	res, err := runCleanup(t, gh, nil, Config{Owner: "nicerobot", Repo: "repo1", Days: 30, Keep: 0})
	require.NoError(t, err)
	assert.Len(t, gh.deletes, 3)
	assert.Equal(t, repo.CreatedBefore("2025-01-02"), gh.lastBefore)
	assert.Equal(t, 3, res.Deleted)
	assert.Equal(t, 0, res.Kept)
	assert.Equal(t, 1, res.ReposScanned)
	require.Len(t, res.Repos, 1)
	assert.Equal(t, RepoResult{Name: "repo1", Deleted: 3, Kept: 0}, res.Repos[0])
}

func TestKeepsMinimumPerWorkflow(t *testing.T) {
	gh := &fakeGH{runs: map[string][]github.WorkflowRun{
		"repo1": {
			mkRun(1, 10, "2025-01-01T00:00:00Z"),
			mkRun(2, 10, "2025-01-02T00:00:00Z"),
			mkRun(3, 10, "2025-01-03T00:00:00Z"),
			mkRun(4, 20, "2025-01-01T00:00:00Z"),
			mkRun(5, 20, "2025-01-02T00:00:00Z"),
		},
	}}
	res, err := runCleanup(t, gh, nil, Config{Owner: "nicerobot", Repo: "repo1", Days: 30, Keep: 2})
	require.NoError(t, err)
	require.Len(t, gh.deletes, 1)
	assert.Equal(t, deleteCall{"nicerobot", "repo1", 1}, gh.deletes[0])
	assert.Equal(t, 1, res.Deleted)
	assert.Equal(t, 4, res.Kept)
}

func TestDryRunDoesNotDelete(t *testing.T) {
	gh := &fakeGH{runs: map[string][]github.WorkflowRun{
		"repo1": {mkRun(1, 1, "2025-01-01T00:00:00Z"), mkRun(2, 1, "2025-01-02T00:00:00Z")},
	}}
	res, err := runCleanup(t, gh, nil, Config{Owner: "nicerobot", Repo: "repo1", Days: 30, Keep: 0, DryRun: true})
	require.NoError(t, err)
	assert.Empty(t, gh.deletes)
	assert.True(t, res.DryRun)
	assert.Equal(t, 2, res.Deleted)
}

func TestSingleRepoMode(t *testing.T) {
	gh := &fakeGH{runs: map[string][]github.WorkflowRun{"target": {mkRun(1, 1, "2025-01-01T00:00:00Z")}}}
	_, err := runCleanup(t, gh, nil, Config{Owner: "nicerobot", Repo: "target", Days: 30, Keep: 0})
	require.NoError(t, err)
	assert.False(t, gh.listReposCalled)
	require.Len(t, gh.deletes, 1)
	assert.Equal(t, deleteCall{"nicerobot", "target", 1}, gh.deletes[0])
}

func TestAllReposMode(t *testing.T) {
	gh := &fakeGH{
		repos: []github.Repository{mkRepo("repo2"), mkRepo("repo1")},
		runs: map[string][]github.WorkflowRun{
			"repo1": {mkRun(1, 1, "2025-01-01T00:00:00Z")},
			"repo2": {mkRun(2, 1, "2025-01-01T00:00:00Z")},
		},
	}
	res, err := runCleanup(t, gh, nil, Config{Owner: "nicerobot", Days: 30, Keep: 0})
	require.NoError(t, err)
	assert.True(t, gh.listReposCalled)
	require.Len(t, gh.deletes, 2)
	// sorted: repo1 before repo2
	assert.Equal(t, deleteCall{"nicerobot", "repo1", 1}, gh.deletes[0])
	assert.Equal(t, deleteCall{"nicerobot", "repo2", 2}, gh.deletes[1])
	assert.Equal(t, 2, res.ReposScanned)
}

func TestNoOldRunsNothingDeleted(t *testing.T) {
	gh := &fakeGH{runs: map[string][]github.WorkflowRun{"repo1": {}}}
	res, err := runCleanup(t, gh, nil, Config{Owner: "nicerobot", Repo: "repo1", Days: 30, Keep: 5})
	require.NoError(t, err)
	assert.Empty(t, gh.deletes)
	assert.Empty(t, res.Repos)
}

func TestEmptyRepoSkippedWhenAllKept(t *testing.T) {
	gh := &fakeGH{
		repos: []github.Repository{mkRepo("empty")},
		runs:  map[string][]github.WorkflowRun{"empty": {mkRun(1, 1, "2025-01-01T00:00:00Z")}},
	}
	res, err := runCleanup(t, gh, nil, Config{Owner: "nicerobot", Days: 30, Keep: 5})
	require.NoError(t, err)
	assert.Empty(t, gh.deletes)
	assert.Equal(t, 1, res.ReposScanned)
	assert.Equal(t, 0, res.Deleted)
	assert.Empty(t, res.Repos)
}

func TestAutoDetectsCurrentRepo(t *testing.T) {
	gh := &fakeGH{runs: map[string][]github.WorkflowRun{"widget": {mkRun(1, 1, "2025-01-01T00:00:00Z")}}}
	_, err := runCleanup(t, gh, map[string]string{"GITHUB_REPOSITORY": "acme/widget"}, Config{Days: 30, Keep: 0})
	require.NoError(t, err)
	assert.False(t, gh.listReposCalled)
	require.Len(t, gh.deletes, 1)
	assert.Equal(t, deleteCall{"acme", "widget", 1}, gh.deletes[0])
}

func TestMissingOwnerWithoutEnv(t *testing.T) {
	gh := &fakeGH{}
	_, err := runCleanup(t, gh, map[string]string{"GITHUB_REPOSITORY": ""}, Config{Days: 30, Keep: 0})
	require.ErrorIs(t, err, constants.ErrNoTarget)
	assert.Empty(t, gh.deletes)
}

func TestMalformedEnvWithoutSlash(t *testing.T) {
	gh := &fakeGH{}
	_, err := runCleanup(t, gh, map[string]string{"GITHUB_REPOSITORY": "noslash"}, Config{Days: 30, Keep: 0})
	require.ErrorIs(t, err, constants.ErrNoTarget)
}

func TestListReposError(t *testing.T) {
	gh := &fakeGH{listReposErr: errors.New("boom")}
	_, err := runCleanup(t, gh, nil, Config{Owner: "nicerobot", Days: 30, Keep: 0})
	require.Error(t, err)
}

func TestListRunsError(t *testing.T) {
	gh := &fakeGH{listRunsErr: errors.New("boom")}
	_, err := runCleanup(t, gh, nil, Config{Owner: "nicerobot", Repo: "repo1", Days: 30, Keep: 0})
	require.Error(t, err)
}

func TestDeleteError(t *testing.T) {
	gh := &fakeGH{
		deleteErr: errors.New("boom"),
		runs:      map[string][]github.WorkflowRun{"repo1": {mkRun(1, 1, "2025-01-01T00:00:00Z")}},
	}
	_, err := runCleanup(t, gh, nil, Config{Owner: "nicerobot", Repo: "repo1", Days: 30, Keep: 0})
	require.Error(t, err)
}

// TestOSDeps exercises the production wiring: a present token yields a client,
// an absent one surfaces ErrNoAuth.
func TestOSDeps(t *testing.T) {
	t.Setenv("GH_TOKEN", "tok")
	d, err := osDeps()
	require.NoError(t, err)
	assert.NotNil(t, d.github)
	assert.NotNil(t, d.getenv)
	assert.NotNil(t, d.now)

	t.Setenv("GH_TOKEN", "")
	_, err = osDeps()
	require.ErrorIs(t, err, constants.ErrNoAuth)
}

func TestRunBuildsDepsError(t *testing.T) {
	orig := deps
	t.Cleanup(func() { deps = orig })
	deps = func() (dependencies, error) { return dependencies{}, constants.ErrNoAuth.With(nil) }
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	_, err := Run(context.Background(), logger, Config{Owner: "x", Days: 30, Keep: 0})
	require.ErrorIs(t, err, constants.ErrNoAuth)
}
