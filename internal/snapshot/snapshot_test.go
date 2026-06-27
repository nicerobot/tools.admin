package snapshot_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicerobot/tools.admin/internal/adminerr"
	"github.com/nicerobot/tools.admin/internal/domain"
	"github.com/nicerobot/tools.admin/internal/githubmodel"
	"github.com/nicerobot/tools.admin/internal/overrides"
	"github.com/nicerobot/tools.admin/internal/settings"
	"github.com/nicerobot/tools.admin/internal/snapshot"
)

type fakeGH struct {
	accountType domain.AccountType
	repos       []githubmodel.Repository
	exists      map[string]bool
	accountErr  error
	listErr     error
	existsErr   error
	existsCalls []domain.RepoName
}

func (f *fakeGH) GetAccountType(domain.Owner) (domain.AccountType, error) {
	return f.accountType, f.accountErr
}

func (f *fakeGH) ListRepos(domain.Owner) ([]githubmodel.Repository, error) {
	return f.repos, f.listErr
}

func (f *fakeGH) RepoExists(_ domain.Owner, n domain.RepoName) (bool, error) {
	f.existsCalls = append(f.existsCalls, n)
	if f.existsErr != nil {
		return false, f.existsErr
	}
	return f.exists[string(n)], nil
}

type harness struct {
	gh        *fakeGH
	existing  []string
	listErr   error
	loadErr   error
	writeErr  error
	removeErr error
	out       bytes.Buffer
	written   []overrides.File
	removed   []string
}

func (h *harness) deps() snapshot.Deps {
	return snapshot.Deps{
		GitHub: h.gh,
		Load: func(domain.SettingsPath) (settings.OrgSettings, error) {
			return settings.Defaults(), h.loadErr
		},
		Write: func(f overrides.File, dir overrides.ReposDir) (overrides.OutFile, error) {
			if h.writeErr != nil {
				return "", h.writeErr
			}
			h.written = append(h.written, f)
			return overrides.OutFile(string(dir) + "/" + string(f.Name) + ".yml"), nil
		},
		ListExisting: func(overrides.ReposDir) ([]string, error) {
			return h.existing, h.listErr
		},
		Remove: func(path string) error {
			if h.removeErr != nil {
				return h.removeErr
			}
			h.removed = append(h.removed, path)
			return nil
		},
		Out: &h.out,
	}
}

func (h *harness) run(owner domain.Owner) error {
	return snapshot.Run(h.deps(), owner, ".github")
}

func matchingRepo(name string) githubmodel.Repository {
	return githubmodel.Repository{
		Name:                name,
		Private:             true,
		DefaultBranch:       "main",
		AllowSquashMerge:    true,
		AllowMergeCommit:    true,
		AllowRebaseMerge:    true,
		DeleteBranchOnMerge: true,
	}
}

func TestWritesOverrideFiles(t *testing.T) {
	repo1 := matchingRepo("repo1")
	repo1.HasIssues = true
	h := &harness{gh: &fakeGH{accountType: "User", repos: []githubmodel.Repository{repo1, matchingRepo("repo2")}}}
	require.NoError(t, h.run("nicerobot"))
	require.Len(t, h.written, 2)
	require.NotNil(t, h.written[0].Repository.HasIssues)
	assert.True(t, *h.written[0].Repository.HasIssues)
	assert.Nil(t, h.written[1].Repository.HasIssues)
	assert.Contains(t, h.out.String(), "  wrote .github/repos/repo1.yml\n")
}

func TestRemovesStaleVerified404(t *testing.T) {
	h := &harness{
		gh:       &fakeGH{accountType: "User", repos: []githubmodel.Repository{matchingRepo("alive")}, exists: map[string]bool{"dead": false}},
		existing: []string{"alive", "dead"},
	}
	require.NoError(t, h.run("nicerobot"))
	assert.Equal(t, []domain.RepoName{"dead"}, h.gh.existsCalls)
	assert.Equal(t, []string{".github/repos/dead.yml"}, h.removed)
	assert.Contains(t, h.out.String(), "  removing .github/repos/dead.yml (repo no longer exists)\n")
}

func TestAbortsWhenStaleRepoExists(t *testing.T) {
	h := &harness{
		gh:       &fakeGH{accountType: "User", repos: []githubmodel.Repository{matchingRepo("visible")}, exists: map[string]bool{"hidden": true}},
		existing: []string{"visible", "hidden"},
	}
	err := h.run("nicerobot")
	require.ErrorIs(t, err, adminerr.ErrStaleRepoExists)
	assert.Empty(t, h.written) // aborts before any write
	assert.Empty(t, h.removed)
}

func TestAbortsOnVerificationError(t *testing.T) {
	h := &harness{
		gh:       &fakeGH{accountType: "User", repos: []githubmodel.Repository{matchingRepo("alive")}, existsErr: errors.New("server error")},
		existing: []string{"alive", "maybe"},
	}
	err := h.run("nicerobot")
	require.Error(t, err)
	assert.NotErrorIs(t, err, adminerr.ErrStaleRepoExists)
	assert.Empty(t, h.written)
}

func TestNoStaleSkipsVerification(t *testing.T) {
	h := &harness{
		gh:       &fakeGH{accountType: "User", repos: []githubmodel.Repository{matchingRepo("repo1")}},
		existing: []string{"repo1"},
	}
	require.NoError(t, h.run("nicerobot"))
	assert.Empty(t, h.gh.existsCalls)
	assert.Empty(t, h.removed)
	require.Len(t, h.written, 1)
}

func TestOrgCommentSource(t *testing.T) {
	h := &harness{gh: &fakeGH{accountType: domain.AccountTypeOrganization, repos: []githubmodel.Repository{matchingRepo("p")}}}
	require.NoError(t, h.run("myorg"))
	assert.Equal(t, domain.CommentSourceOrg, h.written[0].Source)
}

func TestAccountCommentSource(t *testing.T) {
	h := &harness{gh: &fakeGH{accountType: "User", repos: []githubmodel.Repository{matchingRepo("p")}}}
	require.NoError(t, h.run("nicerobot"))
	assert.Equal(t, domain.CommentSourceAccount, h.written[0].Source)
}

func TestForkMarked(t *testing.T) {
	r := matchingRepo("forked")
	r.Fork = true
	h := &harness{gh: &fakeGH{accountType: "User", repos: []githubmodel.Repository{r}}}
	require.NoError(t, h.run("nicerobot"))
	assert.Equal(t, domain.IsFork(true), h.written[0].IsFork)
}

func TestEmptyReposCleansVerifiedStale(t *testing.T) {
	h := &harness{
		gh:       &fakeGH{accountType: "User", exists: map[string]bool{"orphan": false}},
		existing: []string{"orphan"},
	}
	require.NoError(t, h.run("nicerobot"))
	assert.Equal(t, []string{".github/repos/orphan.yml"}, h.removed)
}

func TestLoadError(t *testing.T) {
	h := &harness{gh: &fakeGH{}, loadErr: errors.New("missing")}
	require.Error(t, h.run("nicerobot"))
}

func TestAccountTypeError(t *testing.T) {
	h := &harness{gh: &fakeGH{accountErr: errors.New("boom")}}
	require.Error(t, h.run("nicerobot"))
}

func TestListReposError(t *testing.T) {
	h := &harness{gh: &fakeGH{accountType: "User", listErr: errors.New("boom")}}
	require.Error(t, h.run("nicerobot"))
}

func TestListExistingError(t *testing.T) {
	h := &harness{gh: &fakeGH{accountType: "User", repos: []githubmodel.Repository{matchingRepo("r")}}, listErr: errors.New("boom")}
	require.Error(t, h.run("nicerobot"))
}

func TestWriteError(t *testing.T) {
	h := &harness{gh: &fakeGH{accountType: "User", repos: []githubmodel.Repository{matchingRepo("r")}}, writeErr: errors.New("disk")}
	require.Error(t, h.run("nicerobot"))
}

func TestRemoveError(t *testing.T) {
	h := &harness{
		gh:        &fakeGH{accountType: "User", exists: map[string]bool{"gone": false}},
		existing:  []string{"gone"},
		removeErr: errors.New("perm"),
	}
	err := h.run("nicerobot")
	require.ErrorIs(t, err, adminerr.ErrRemoveFile)
}
