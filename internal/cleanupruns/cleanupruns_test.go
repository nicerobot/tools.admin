package cleanupruns_test

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicerobot/tools.admin/internal/adminerr"
	"github.com/nicerobot/tools.admin/internal/cleanupruns"
	"github.com/nicerobot/tools.admin/internal/domain"
	"github.com/nicerobot/tools.admin/internal/githubmodel"
)

type deleteCall struct {
	owner domain.Owner
	repo  domain.RepoName
	id    domain.RunID
}

type fakeGH struct {
	repos        []githubmodel.Repository
	runs         map[string][]githubmodel.WorkflowRun
	listReposErr error
	listRunsErr  error
	deleteErr    error

	listReposCalled bool
	deletes         []deleteCall
	lastBefore      domain.CreatedBefore
}

func (f *fakeGH) ListRepos(domain.Owner) ([]githubmodel.Repository, error) {
	f.listReposCalled = true
	return f.repos, f.listReposErr
}

func (f *fakeGH) ListWorkflowRuns(_ domain.Owner, r domain.RepoName, b domain.CreatedBefore) ([]githubmodel.WorkflowRun, error) {
	f.lastBefore = b
	if f.listRunsErr != nil {
		return nil, f.listRunsErr
	}
	return f.runs[string(r)], nil
}

func (f *fakeGH) DeleteWorkflowRun(o domain.Owner, r domain.RepoName, id domain.RunID) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deletes = append(f.deletes, deleteCall{o, r, id})
	return nil
}

func mkRun(id, wf int64, created string) githubmodel.WorkflowRun {
	return githubmodel.WorkflowRun{ID: id, WorkflowID: wf, CreatedAt: created, Name: "CI", Status: "completed"}
}

func mkRepo(name string) githubmodel.Repository { return githubmodel.Repository{Name: name} }

func fixedNow() time.Time { return time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC) }

func runCleanup(gh *fakeGH, env map[string]string, o cleanupruns.Options) (string, error) {
	var out bytes.Buffer
	deps := cleanupruns.Deps{
		GitHub: gh,
		Env:    func(k string) string { return env[k] },
		Now:    fixedNow,
		Out:    &out,
	}
	err := cleanupruns.Run(deps, o)
	return out.String(), err
}

func TestDeletesOldRuns(t *testing.T) {
	gh := &fakeGH{runs: map[string][]githubmodel.WorkflowRun{
		"repo1": {mkRun(1, 1, "2025-01-01T00:00:00Z"), mkRun(2, 1, "2025-01-02T00:00:00Z"), mkRun(3, 1, "2025-01-03T00:00:00Z")},
	}}
	out, err := runCleanup(gh, nil, cleanupruns.Options{Owner: "nicerobot", Repo: "repo1", Days: 30, Keep: 0})
	require.NoError(t, err)
	assert.Len(t, gh.deletes, 3)
	assert.Equal(t, domain.CreatedBefore("2025-01-02"), gh.lastBefore)
	assert.Contains(t, out, "nicerobot/repo1: deleting 3, keeping 0")
	assert.Contains(t, out, "\nSummary: 1 repos scanned, 3 runs deleted, 0 runs kept\n")
}

func TestKeepsMinimumPerWorkflow(t *testing.T) {
	gh := &fakeGH{runs: map[string][]githubmodel.WorkflowRun{
		"repo1": {
			mkRun(1, 10, "2025-01-01T00:00:00Z"),
			mkRun(2, 10, "2025-01-02T00:00:00Z"),
			mkRun(3, 10, "2025-01-03T00:00:00Z"),
			mkRun(4, 20, "2025-01-01T00:00:00Z"),
			mkRun(5, 20, "2025-01-02T00:00:00Z"),
		},
	}}
	_, err := runCleanup(gh, nil, cleanupruns.Options{Owner: "nicerobot", Repo: "repo1", Days: 30, Keep: 2})
	require.NoError(t, err)
	require.Len(t, gh.deletes, 1)
	assert.Equal(t, deleteCall{"nicerobot", "repo1", 1}, gh.deletes[0])
}

func TestDryRunDoesNotDelete(t *testing.T) {
	gh := &fakeGH{runs: map[string][]githubmodel.WorkflowRun{
		"repo1": {mkRun(1, 1, "2025-01-01T00:00:00Z"), mkRun(2, 1, "2025-01-02T00:00:00Z")},
	}}
	out, err := runCleanup(gh, nil, cleanupruns.Options{Owner: "nicerobot", Repo: "repo1", Days: 30, Keep: 0, DryRun: true})
	require.NoError(t, err)
	assert.Empty(t, gh.deletes)
	assert.Contains(t, out, "[dry-run] would delete   run 1 (CI, 2025-01-01T00:00:00Z)")
	assert.Contains(t, out, "\nSummary: 1 repos scanned, 2 runs would delete, 0 runs kept\n")
}

func TestSingleRepoMode(t *testing.T) {
	gh := &fakeGH{runs: map[string][]githubmodel.WorkflowRun{"target": {mkRun(1, 1, "2025-01-01T00:00:00Z")}}}
	_, err := runCleanup(gh, nil, cleanupruns.Options{Owner: "nicerobot", Repo: "target", Days: 30, Keep: 0})
	require.NoError(t, err)
	assert.False(t, gh.listReposCalled)
	require.Len(t, gh.deletes, 1)
	assert.Equal(t, deleteCall{"nicerobot", "target", 1}, gh.deletes[0])
}

func TestAllReposMode(t *testing.T) {
	gh := &fakeGH{
		repos: []githubmodel.Repository{mkRepo("repo2"), mkRepo("repo1")},
		runs: map[string][]githubmodel.WorkflowRun{
			"repo1": {mkRun(1, 1, "2025-01-01T00:00:00Z")},
			"repo2": {mkRun(2, 1, "2025-01-01T00:00:00Z")},
		},
	}
	_, err := runCleanup(gh, nil, cleanupruns.Options{Owner: "nicerobot", Days: 30, Keep: 0})
	require.NoError(t, err)
	assert.True(t, gh.listReposCalled)
	require.Len(t, gh.deletes, 2)
	// sorted: repo1 before repo2
	assert.Equal(t, deleteCall{"nicerobot", "repo1", 1}, gh.deletes[0])
	assert.Equal(t, deleteCall{"nicerobot", "repo2", 2}, gh.deletes[1])
}

func TestNoOldRunsNothingDeleted(t *testing.T) {
	gh := &fakeGH{runs: map[string][]githubmodel.WorkflowRun{"repo1": {}}}
	_, err := runCleanup(gh, nil, cleanupruns.Options{Owner: "nicerobot", Repo: "repo1", Days: 30, Keep: 5})
	require.NoError(t, err)
	assert.Empty(t, gh.deletes)
}

func TestEmptyRepoSkippedWhenAllKept(t *testing.T) {
	gh := &fakeGH{
		repos: []githubmodel.Repository{mkRepo("empty")},
		runs:  map[string][]githubmodel.WorkflowRun{"empty": {mkRun(1, 1, "2025-01-01T00:00:00Z")}},
	}
	out, err := runCleanup(gh, nil, cleanupruns.Options{Owner: "nicerobot", Days: 30, Keep: 5})
	require.NoError(t, err)
	assert.Empty(t, gh.deletes)
	assert.Contains(t, out, "\nSummary: 1 repos scanned, 0 runs deleted, 0 runs kept\n")
}

func TestAutoDetectsCurrentRepo(t *testing.T) {
	gh := &fakeGH{runs: map[string][]githubmodel.WorkflowRun{"widget": {mkRun(1, 1, "2025-01-01T00:00:00Z")}}}
	_, err := runCleanup(gh, map[string]string{"GITHUB_REPOSITORY": "acme/widget"}, cleanupruns.Options{Days: 30, Keep: 0})
	require.NoError(t, err)
	assert.False(t, gh.listReposCalled)
	require.Len(t, gh.deletes, 1)
	assert.Equal(t, deleteCall{"acme", "widget", 1}, gh.deletes[0])
}

func TestMissingOwnerWithoutEnv(t *testing.T) {
	gh := &fakeGH{}
	_, err := runCleanup(gh, map[string]string{"GITHUB_REPOSITORY": ""}, cleanupruns.Options{Days: 30, Keep: 0})
	require.ErrorIs(t, err, adminerr.ErrNoTarget)
	assert.Empty(t, gh.deletes)
}

func TestMalformedEnvWithoutSlash(t *testing.T) {
	gh := &fakeGH{}
	_, err := runCleanup(gh, map[string]string{"GITHUB_REPOSITORY": "noslash"}, cleanupruns.Options{Days: 30, Keep: 0})
	require.ErrorIs(t, err, adminerr.ErrNoTarget)
}

func TestListReposError(t *testing.T) {
	gh := &fakeGH{listReposErr: errors.New("boom")}
	_, err := runCleanup(gh, nil, cleanupruns.Options{Owner: "nicerobot", Days: 30, Keep: 0})
	require.Error(t, err)
}

func TestListRunsError(t *testing.T) {
	gh := &fakeGH{listRunsErr: errors.New("boom")}
	_, err := runCleanup(gh, nil, cleanupruns.Options{Owner: "nicerobot", Repo: "repo1", Days: 30, Keep: 0})
	require.Error(t, err)
}

func TestDeleteError(t *testing.T) {
	gh := &fakeGH{
		deleteErr: errors.New("boom"),
		runs:      map[string][]githubmodel.WorkflowRun{"repo1": {mkRun(1, 1, "2025-01-01T00:00:00Z")}},
	}
	_, err := runCleanup(gh, nil, cleanupruns.Options{Owner: "nicerobot", Repo: "repo1", Days: 30, Keep: 0})
	require.Error(t, err)
}
