package overrides_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicerobot/tools.admin/internal/adminerr"
	"github.com/nicerobot/tools.admin/internal/domain"
	"github.com/nicerobot/tools.admin/internal/githubmodel"
	"github.com/nicerobot/tools.admin/internal/overrides"
	"github.com/nicerobot/tools.admin/internal/settings"
)

func sp(s string) *string                       { return &s }
func bp(b bool) *bool                           { return &b }
func vp(v domain.Visibility) *domain.Visibility { return &v }

func defaults() settings.RepositoryDefaults {
	return settings.RepositoryDefaults{
		DefaultBranch:       "main",
		Visibility:          domain.VisibilityPrivate,
		AllowSquashMerge:    true,
		AllowMergeCommit:    true,
		AllowRebaseMerge:    true,
		DeleteBranchOnMerge: true,
	}
}

// matchingRepo returns a repo whose every field equals the defaults, so it
// produces no overrides unless a field is changed by the caller.
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

func TestComputeNoOverridesWhenMatching(t *testing.T) {
	f := overrides.Compute(matchingRepo("boring"), defaults(), "nicerobot", domain.CommentSourceAccount)
	assert.Equal(t, overrides.Repository{}, f.Repository)
	assert.Equal(t, domain.IsFork(false), f.IsFork)
	assert.Equal(t, domain.Owner("nicerobot"), f.Owner)
	assert.Equal(t, domain.RepoName("boring"), f.Name)
}

func TestComputeStringFields(t *testing.T) {
	r := matchingRepo("test")
	r.Description = "About nicerobot"
	r.Homepage = "https://example.com"
	r.Private = false // → visibility public (differs)
	r.DefaultBranch = "master"
	f := overrides.Compute(r, defaults(), "nicerobot", domain.CommentSourceAccount)
	require.NotNil(t, f.Repository.Description)
	assert.Equal(t, "About nicerobot", *f.Repository.Description)
	require.NotNil(t, f.Repository.Homepage)
	assert.Equal(t, "https://example.com", *f.Repository.Homepage)
	require.NotNil(t, f.Repository.Visibility)
	assert.Equal(t, domain.VisibilityPublic, *f.Repository.Visibility)
	require.NotNil(t, f.Repository.DefaultBranch)
	assert.Equal(t, "master", *f.Repository.DefaultBranch)
}

func TestComputeEmptyStringsExcluded(t *testing.T) {
	r := matchingRepo("test")
	r.Description = ""
	r.Homepage = ""
	f := overrides.Compute(r, defaults(), "nicerobot", domain.CommentSourceAccount)
	assert.Nil(t, f.Repository.Description)
	assert.Nil(t, f.Repository.Homepage)
}

func TestComputeBoolDiffsAndArchived(t *testing.T) {
	r := matchingRepo("admin")
	r.HasIssues = true
	r.HasProjects = true
	r.HasWiki = true
	r.HasDiscussions = true
	r.IsTemplate = true
	r.AllowSquashMerge = false
	r.AllowMergeCommit = false
	r.AllowRebaseMerge = false
	r.AllowAutoMerge = true
	r.DeleteBranchOnMerge = false
	r.Archived = true
	f := overrides.Compute(r, defaults(), "nicerobot", domain.CommentSourceAccount).Repository
	require.NotNil(t, f.HasIssues)
	assert.True(t, *f.HasIssues)
	require.NotNil(t, f.DeleteBranchOnMerge)
	assert.False(t, *f.DeleteBranchOnMerge)
	require.NotNil(t, f.Archived)
	assert.True(t, *f.Archived)
}

func TestComputeArchivedFalseExcluded(t *testing.T) {
	r := matchingRepo("test")
	r.Archived = false
	f := overrides.Compute(r, defaults(), "nicerobot", domain.CommentSourceAccount)
	assert.Nil(t, f.Repository.Archived)
}

func TestComputeForkFlag(t *testing.T) {
	r := matchingRepo("forked")
	r.Fork = true
	f := overrides.Compute(r, defaults(), "nicerobot", domain.CommentSourceAccount)
	assert.Equal(t, domain.IsFork(true), f.IsFork)
}

func TestRenderEmptyRepository(t *testing.T) {
	f := overrides.File{Owner: "nicerobot", Name: "boring", Source: domain.CommentSourceAccount}
	want := "# nicerobot/boring — overrides from account defaults\n\nrepository: {}\n"
	assert.Equal(t, want, f.Render())
}

func TestRenderForkEmpty(t *testing.T) {
	f := overrides.File{Owner: "nicerobot", Name: "forked", Source: domain.CommentSourceAccount, IsFork: true}
	want := "# nicerobot/forked — overrides from account defaults\n\n_fork: true\n\nrepository: {}\n"
	assert.Equal(t, want, f.Render())
}

func TestRenderForkWithOverrides(t *testing.T) {
	f := overrides.File{
		Owner: "nicerobot", Name: "forked", Source: domain.CommentSourceAccount, IsFork: true,
		Repository: overrides.Repository{HasIssues: bp(true)},
	}
	want := "# nicerobot/forked — overrides from account defaults\n\n_fork: true\n\nrepository:\n  has_issues: true\n"
	assert.Equal(t, want, f.Render())
}

// TestRenderAdminGolden reproduces the real admin.yml byte-for-byte.
func TestRenderAdminGolden(t *testing.T) {
	f := overrides.File{
		Owner: "nicerobot", Name: "admin", Source: domain.CommentSourceAccount,
		Repository: overrides.Repository{
			DefaultBranch:       sp("main"),
			HasIssues:           bp(true),
			HasProjects:         bp(true),
			HasWiki:             bp(true),
			DeleteBranchOnMerge: bp(false),
		},
	}
	want := "# nicerobot/admin — overrides from account defaults\n" +
		"\n" +
		"repository:\n" +
		"  default_branch: main\n" +
		"  has_issues: true\n" +
		"  has_projects: true\n" +
		"  has_wiki: true\n" +
		"  delete_branch_on_merge: false\n"
	assert.Equal(t, want, f.Render())
}

func TestRenderQuotedStringsAndVisibility(t *testing.T) {
	f := overrides.File{
		Owner: "nicerobot", Name: "year", Source: domain.CommentSourceOrg,
		Repository: overrides.Repository{
			Description:   sp("About nicerobot"),
			Homepage:      sp("https://nicerobot.github.io/year?i=2024"),
			Visibility:    vp(domain.VisibilityPublic),
			DefaultBranch: sp("master"),
			Archived:      bp(true),
		},
	}
	want := "# nicerobot/year — overrides from org defaults\n" +
		"\n" +
		"repository:\n" +
		`  description: "About nicerobot"` + "\n" +
		`  homepage: "https://nicerobot.github.io/year?i=2024"` + "\n" +
		"  visibility: public\n" +
		"  default_branch: master\n" +
		"  archived: true\n"
	assert.Equal(t, want, f.Render())
}

// TestComputeRenderProfileGolden integrates Compute → Render for the real
// nicerobot profile repo.
func TestComputeRenderProfileGolden(t *testing.T) {
	r := matchingRepo("nicerobot")
	r.Description = "About nicerobot"
	r.Private = false
	r.DefaultBranch = "master"
	f := overrides.Compute(r, defaults(), "nicerobot", domain.CommentSourceAccount)
	want := "# nicerobot/nicerobot — overrides from account defaults\n" +
		"\n" +
		"repository:\n" +
		`  description: "About nicerobot"` + "\n" +
		"  visibility: public\n" +
		"  default_branch: master\n"
	assert.Equal(t, want, f.Render())
}

func TestWriteSuccess(t *testing.T) {
	var gotDir, gotName string
	var gotData []byte
	mkdir := func(p string) error { gotDir = p; return nil }
	write := func(name string, data []byte) error { gotName = name; gotData = data; return nil }
	f := overrides.File{Owner: "o", Name: "test", Source: domain.CommentSourceAccount}
	out, err := overrides.Write(f, "settings/repos", mkdir, write)
	require.NoError(t, err)
	assert.Equal(t, "settings/repos", gotDir)
	assert.Equal(t, "settings/repos/test.yml", gotName)
	assert.Equal(t, overrides.OutFile("settings/repos/test.yml"), out)
	assert.Equal(t, f.Render(), string(gotData))
}

func TestWriteMkdirError(t *testing.T) {
	mkdir := func(string) error { return errors.New("denied") }
	write := func(string, []byte) error { return nil }
	_, err := overrides.Write(overrides.File{Name: "x"}, "d", mkdir, write)
	require.ErrorIs(t, err, adminerr.ErrWriteFile)
}

func TestWriteFileError(t *testing.T) {
	mkdir := func(string) error { return nil }
	write := func(string, []byte) error { return errors.New("disk full") }
	_, err := overrides.Write(overrides.File{Name: "x"}, "d", mkdir, write)
	require.ErrorIs(t, err, adminerr.ErrWriteFile)
}

func TestListExistingStems(t *testing.T) {
	glob := func(pattern string) ([]string, error) {
		assert.Equal(t, "repos/*.yml", pattern)
		return []string{"repos/admin.yml", "repos/site-example.com.yml"}, nil
	}
	stems, err := overrides.ListExisting("repos", glob)
	require.NoError(t, err)
	assert.Equal(t, []string{"admin", "site-example.com"}, stems)
}

func TestListExistingGlobError(t *testing.T) {
	glob := func(string) ([]string, error) { return nil, errors.New("bad pattern") }
	_, err := overrides.ListExisting("repos", glob)
	require.ErrorIs(t, err, adminerr.ErrListRepoFiles)
}
