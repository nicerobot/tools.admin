package settings_test

import (
	"errors"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicerobot/tools.admin/internal/constants"
	"github.com/nicerobot/tools.admin/internal/repo"
	"github.com/nicerobot/tools.admin/internal/settings"
)

func reader(data string, err error) settings.ReadFileFunc {
	return func(string) ([]byte, error) {
		if err != nil {
			return nil, err
		}
		return []byte(data), nil
	}
}

func TestDefaults(t *testing.T) {
	d := settings.Defaults().Repository
	assert.Equal(t, "main", d.DefaultBranch)
	assert.Equal(t, repo.VisibilityPrivate, d.Visibility)
	assert.False(t, d.HasIssues)
	assert.True(t, d.CanSquashMerge)
	assert.True(t, d.CanMergeCommit)
	assert.True(t, d.CanRebaseMerge)
	assert.True(t, d.ShouldDeleteBranchOnMerge)
}

func TestLoadEmptyKeepsDefaults(t *testing.T) {
	s, err := settings.Load(reader("", nil), ".github")
	require.NoError(t, err)
	assert.Equal(t, "main", s.Repository.DefaultBranch)
	assert.Equal(t, repo.VisibilityPrivate, s.Repository.Visibility)
	assert.True(t, s.Repository.ShouldDeleteBranchOnMerge)
}

func TestLoadPartialOverridesOnlyPresentKeys(t *testing.T) {
	s, err := settings.Load(reader("repository:\n  visibility: public\n", nil), ".github")
	require.NoError(t, err)
	assert.Equal(t, repo.VisibilityPublic, s.Repository.Visibility)
	assert.Equal(t, "main", s.Repository.DefaultBranch) // default retained
}

func TestLoadFullRealFormat(t *testing.T) {
	data := "repository:\n" +
		"  default_branch: main\n" +
		"  visibility: private\n" +
		"  has_issues: false\n" +
		"  has_projects: false\n" +
		"  has_wiki: false\n" +
		"  has_discussions: false\n" +
		"  is_template: false\n" +
		"  allow_squash_merge: true\n" +
		"  allow_merge_commit: true\n" +
		"  allow_rebase_merge: true\n" +
		"  allow_auto_merge: false\n" +
		"  delete_branch_on_merge: true\n" +
		"labels:\n" +
		"  - name: bug\n" +
		"    color: d73a4a\n"
	s, err := settings.Load(reader(data, nil), ".github")
	require.NoError(t, err)
	assert.Equal(t, "main", s.Repository.DefaultBranch)
	assert.True(t, s.Repository.ShouldDeleteBranchOnMerge)
	assert.False(t, s.Repository.CanAutoMerge)
}

func TestLoadMissingFileNotExist(t *testing.T) {
	_, err := settings.Load(reader("", fs.ErrNotExist), ".github")
	require.ErrorIs(t, err, constants.ErrSettingsNotFound)
}

func TestLoadReadErrorOther(t *testing.T) {
	other := errors.New("permission denied")
	_, err := settings.Load(reader("", other), ".github")
	require.ErrorIs(t, err, constants.ErrSettingsNotFound)
	require.ErrorIs(t, err, other)
}

func TestLoadInvalidYAML(t *testing.T) {
	_, err := settings.Load(reader("repository: [unterminated\n", nil), ".github")
	require.ErrorIs(t, err, constants.ErrInvalidSettings)
}
