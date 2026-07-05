package overrides_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicerobot/tools.admin/internal/constants"
	"github.com/nicerobot/tools.admin/internal/github"
	"github.com/nicerobot/tools.admin/internal/overrides"
	"github.com/nicerobot/tools.admin/internal/repo"
	"github.com/nicerobot/tools.admin/internal/settings"
)

func sp(s string) *string                   { return &s }
func bp(b bool) *bool                       { return &b }
func vp(v repo.Visibility) *repo.Visibility { return &v }

func defaults() settings.RepositoryDefaults {
	return settings.RepositoryDefaults{
		DefaultBranch:             "main",
		Visibility:                repo.VisibilityPrivate,
		CanSquashMerge:            true,
		CanMergeCommit:            true,
		CanRebaseMerge:            true,
		ShouldDeleteBranchOnMerge: true,
	}
}

// matchingRepo returns a repo whose every field equals the defaults, so it
// produces no overrides unless a field is changed by the caller.
func matchingRepo(name string) github.Repository {
	return github.Repository{
		Name:                      name,
		IsPrivate:                 true,
		DefaultBranch:             "main",
		CanSquashMerge:            true,
		CanMergeCommit:            true,
		CanRebaseMerge:            true,
		ShouldDeleteBranchOnMerge: true,
	}
}

func TestComputeNoOverridesWhenMatching(t *testing.T) {
	f := overrides.Compute(matchingRepo("boring"), defaults(), "nicerobot", repo.CommentSourceAccount)
	assert.Equal(t, overrides.Repository{}, f.Repository)
	assert.Equal(t, repo.IsFork(false), f.IsFork)
	assert.Equal(t, repo.Owner("nicerobot"), f.Owner)
	assert.Equal(t, repo.Name("boring"), f.Name)
}

func TestComputeStringFields(t *testing.T) {
	r := matchingRepo("test")
	r.Description = "About nicerobot"
	r.Homepage = "https://example.com"
	r.IsPrivate = false // → visibility public (differs)
	r.DefaultBranch = "master"
	f := overrides.Compute(r, defaults(), "nicerobot", repo.CommentSourceAccount)
	require.NotNil(t, f.Repository.Description)
	assert.Equal(t, "About nicerobot", *f.Repository.Description)
	require.NotNil(t, f.Repository.Homepage)
	assert.Equal(t, "https://example.com", *f.Repository.Homepage)
	require.NotNil(t, f.Repository.Visibility)
	assert.Equal(t, repo.VisibilityPublic, *f.Repository.Visibility)
	require.NotNil(t, f.Repository.DefaultBranch)
	assert.Equal(t, "master", *f.Repository.DefaultBranch)
}

func TestComputeEmptyStringsExcluded(t *testing.T) {
	r := matchingRepo("test")
	r.Description = ""
	r.Homepage = ""
	f := overrides.Compute(r, defaults(), "nicerobot", repo.CommentSourceAccount)
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
	r.CanSquashMerge = false
	r.CanMergeCommit = false
	r.CanRebaseMerge = false
	r.CanAutoMerge = true
	r.ShouldDeleteBranchOnMerge = false
	r.IsArchived = true
	f := overrides.Compute(r, defaults(), "nicerobot", repo.CommentSourceAccount).Repository
	require.NotNil(t, f.HasIssues)
	assert.True(t, *f.HasIssues)
	require.NotNil(t, f.ShouldDeleteBranchOnMerge)
	assert.False(t, *f.ShouldDeleteBranchOnMerge)
	require.NotNil(t, f.IsArchived)
	assert.True(t, *f.IsArchived)
}

func TestComputeArchivedFalseExcluded(t *testing.T) {
	r := matchingRepo("test")
	r.IsArchived = false
	f := overrides.Compute(r, defaults(), "nicerobot", repo.CommentSourceAccount)
	assert.Nil(t, f.Repository.IsArchived)
}

func TestComputeForkFlag(t *testing.T) {
	r := matchingRepo("forked")
	r.IsFork = true
	f := overrides.Compute(r, defaults(), "nicerobot", repo.CommentSourceAccount)
	assert.Equal(t, repo.IsFork(true), f.IsFork)
}

func TestRenderEmptyRepository(t *testing.T) {
	f := overrides.File{Owner: "nicerobot", Name: "boring", Source: repo.CommentSourceAccount}
	want := "# nicerobot/boring — overrides from account defaults\n\nrepository: {}\n"
	assert.Equal(t, want, f.Render())
}

func TestRenderForkEmpty(t *testing.T) {
	f := overrides.File{Owner: "nicerobot", Name: "forked", Source: repo.CommentSourceAccount, IsFork: true}
	want := "# nicerobot/forked — overrides from account defaults\n\n_fork: true\n\nrepository: {}\n"
	assert.Equal(t, want, f.Render())
}

func TestRenderForkWithOverrides(t *testing.T) {
	f := overrides.File{
		Owner: "nicerobot", Name: "forked", Source: repo.CommentSourceAccount, IsFork: true,
		Repository: overrides.Repository{HasIssues: bp(true)},
	}
	want := "# nicerobot/forked — overrides from account defaults\n\n_fork: true\n\nrepository:\n  has_issues: true\n"
	assert.Equal(t, want, f.Render())
}

// TestRenderAdminGolden reproduces the real admin.yml byte-for-byte.
func TestRenderAdminGolden(t *testing.T) {
	f := overrides.File{
		Owner: "nicerobot", Name: "admin", Source: repo.CommentSourceAccount,
		Repository: overrides.Repository{
			DefaultBranch:             sp("main"),
			HasIssues:                 bp(true),
			HasProjects:               bp(true),
			HasWiki:                   bp(true),
			ShouldDeleteBranchOnMerge: bp(false),
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
		Owner: "nicerobot", Name: "year", Source: repo.CommentSourceOrg,
		Repository: overrides.Repository{
			Description:   sp("About nicerobot"),
			Homepage:      sp("https://nicerobot.github.io/year?i=2024"),
			Visibility:    vp(repo.VisibilityPublic),
			DefaultBranch: sp("master"),
			IsArchived:    bp(true),
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
	r.IsPrivate = false
	r.DefaultBranch = "master"
	f := overrides.Compute(r, defaults(), "nicerobot", repo.CommentSourceAccount)
	want := "# nicerobot/nicerobot — overrides from account defaults\n" +
		"\n" +
		"repository:\n" +
		`  description: "About nicerobot"` + "\n" +
		"  visibility: public\n" +
		"  default_branch: master\n"
	assert.Equal(t, want, f.Render())
}

func TestWriteSuccess(t *testing.T) {
	var gotDir overrides.ReposDir
	var gotName overrides.OutFile
	var gotData []byte
	mkdir := func(p overrides.ReposDir) error { gotDir = p; return nil }
	write := func(name overrides.OutFile, data []byte) error { gotName = name; gotData = data; return nil }
	f := overrides.File{Owner: "o", Name: "test", Source: repo.CommentSourceAccount}
	out, err := overrides.Write(f, "settings/repos", mkdir, write)
	require.NoError(t, err)
	assert.Equal(t, overrides.ReposDir("settings/repos"), gotDir)
	assert.Equal(t, overrides.OutFile("settings/repos/test.yml"), gotName)
	assert.Equal(t, overrides.OutFile("settings/repos/test.yml"), out)
	assert.Equal(t, f.Render(), string(gotData))
}

func TestWriteMkdirError(t *testing.T) {
	mkdir := func(overrides.ReposDir) error { return errors.New("denied") }
	write := func(overrides.OutFile, []byte) error { return nil }
	_, err := overrides.Write(overrides.File{Name: "x"}, "d", mkdir, write)
	require.ErrorIs(t, err, constants.ErrWriteFile)
}

func TestWriteFileError(t *testing.T) {
	mkdir := func(overrides.ReposDir) error { return nil }
	write := func(overrides.OutFile, []byte) error { return errors.New("disk full") }
	_, err := overrides.Write(overrides.File{Name: "x"}, "d", mkdir, write)
	require.ErrorIs(t, err, constants.ErrWriteFile)
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
	require.ErrorIs(t, err, constants.ErrListRepoFiles)
}

func TestOSMkdirAndWriteFile(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b")
	require.NoError(t, overrides.OSMkdir(overrides.ReposDir(nested)))

	file := filepath.Join(nested, "x.yml")
	require.NoError(t, overrides.OSWriteFile(overrides.OutFile(file), []byte("hello")))

	data, err := os.ReadFile(file)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(data))
}
