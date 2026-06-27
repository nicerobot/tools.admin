// Package settings loads the org-level safe-settings defaults from
// <settings-path>/settings.yml. Only the repository: defaults block is
// consumed — labels/collaborators are ignored, exactly as the original CLI did
// (they never fed the snapshot output). Defaults are seeded before decode so a
// key absent from the file keeps its documented default, matching the original
// pydantic model semantics.
package settings

import (
	"errors"
	"io/fs"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/nicerobot/tools.admin/internal/constants"
	"github.com/nicerobot/tools.admin/internal/repo"
)

const settingsFile = "settings.yml"

// ReadFileFunc reads a named file; injected so the missing/invalid paths are
// reachable from unit tests without touching a real filesystem.
type ReadFileFunc func(name string) ([]byte, error)

// RepositoryDefaults is the org's default repository configuration that each
// live repo is diffed against. The defaults seeded by Defaults() are retained
// for any key the file omits.
type RepositoryDefaults struct {
	DefaultBranch       string          `yaml:"default_branch"`
	Visibility          repo.Visibility `yaml:"visibility"`
	HasIssues           bool            `yaml:"has_issues"`
	HasProjects         bool            `yaml:"has_projects"`
	HasWiki             bool            `yaml:"has_wiki"`
	HasDiscussions      bool            `yaml:"has_discussions"`
	IsTemplate          bool            `yaml:"is_template"`
	AllowSquashMerge    bool            `yaml:"allow_squash_merge"`
	AllowMergeCommit    bool            `yaml:"allow_merge_commit"`
	AllowRebaseMerge    bool            `yaml:"allow_rebase_merge"`
	AllowAutoMerge      bool            `yaml:"allow_auto_merge"`
	DeleteBranchOnMerge bool            `yaml:"delete_branch_on_merge"`
}

// OrgSettings is the decoded settings.yml; only repository defaults are used.
type OrgSettings struct {
	Repository RepositoryDefaults `yaml:"repository"`
}

// Defaults returns the baseline OrgSettings seeded before decode. These mirror
// the original model defaults so an empty or partial file yields them.
func Defaults() OrgSettings {
	return OrgSettings{
		Repository: RepositoryDefaults{
			DefaultBranch:       "main",
			Visibility:          repo.VisibilityPrivate,
			AllowSquashMerge:    true,
			AllowMergeCommit:    true,
			AllowRebaseMerge:    true,
			DeleteBranchOnMerge: true,
		},
	}
}

// Load reads <path>/settings.yml via read and decodes it over the defaults. A
// missing file yields ErrSettingsNotFound; malformed YAML yields
// ErrInvalidSettings.
func Load(read ReadFileFunc, path repo.SettingsPath) (OrgSettings, error) {
	file := filepath.Join(string(path), settingsFile)
	data, err := read(file)
	if err != nil {
		return OrgSettings{}, notFound(err, file)
	}
	out := Defaults()
	if err := yaml.Unmarshal(data, &out); err != nil {
		return OrgSettings{}, constants.ErrInvalidSettings.With(err, "file", file)
	}
	return out, nil
}

// notFound classifies a read error as the settings-not-found sentinel, keeping
// the original "fatal when missing" contract.
func notFound(err error, file string) error {
	if errors.Is(err, fs.ErrNotExist) {
		return constants.ErrSettingsNotFound.With(nil, "file", file)
	}
	return constants.ErrSettingsNotFound.With(err, "file", file)
}
