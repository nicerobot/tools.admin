// Package overrides computes a repository's settings diff against the org
// defaults and renders it to the repos/<name>.yml format the settings
// snapshot consumes. The render order is fixed to the field declaration order
// so output is byte-for-byte stable across runs.
package overrides

import (
	"fmt"

	"github.com/nicerobot/tools.admin/internal/github"
	"github.com/nicerobot/tools.admin/internal/repo"
	"github.com/nicerobot/tools.admin/internal/settings"
)

// Repository is the set of override fields, each a pointer so "absent" is
// distinct from a zero value. Field order here is the emit order.
type Repository struct {
	Description         *string
	Homepage            *string
	Visibility          *repo.Visibility
	DefaultBranch       *string
	HasIssues           *bool
	HasProjects         *bool
	HasWiki             *bool
	HasDiscussions      *bool
	IsTemplate          *bool
	AllowSquashMerge    *bool
	AllowMergeCommit    *bool
	AllowRebaseMerge    *bool
	AllowAutoMerge      *bool
	DeleteBranchOnMerge *bool
	Archived            *bool
}

// File is a fully-resolved override document for one repository.
type File struct {
	Repository Repository
	Owner      repo.Owner
	Name       repo.Name
	Source     repo.CommentSource
	IsFork     repo.IsFork
}

// Compute diffs repo against defaults, returning the override document. The
// description/homepage are included when non-empty; archived only when true;
// every other field only when it differs from the default.
func Compute(
	gh github.Repository,
	defaults settings.RepositoryDefaults,
	owner repo.Owner,
	source repo.CommentSource,
) File {
	r := applyBools(applyStrings(Repository{}, gh, defaults), gh, defaults)
	return File{
		Owner:      owner,
		Name:       repo.Name(gh.Name),
		Source:     source,
		IsFork:     repo.IsFork(gh.Fork),
		Repository: r,
	}
}

func applyStrings(r Repository, gh github.Repository, d settings.RepositoryDefaults) Repository {
	if gh.Description != "" {
		r.Description = ptr(gh.Description)
	}
	if gh.Homepage != "" {
		r.Homepage = ptr(gh.Homepage)
	}
	if gh.Visibility() != d.Visibility {
		r.Visibility = ptr(gh.Visibility())
	}
	if gh.DefaultBranch != d.DefaultBranch {
		r.DefaultBranch = ptr(gh.DefaultBranch)
	}
	return r
}

func applyBools(r Repository, gh github.Repository, d settings.RepositoryDefaults) Repository {
	r.HasIssues = boolDiff(gh.HasIssues, d.HasIssues)
	r.HasProjects = boolDiff(gh.HasProjects, d.HasProjects)
	r.HasWiki = boolDiff(gh.HasWiki, d.HasWiki)
	r.HasDiscussions = boolDiff(gh.HasDiscussions, d.HasDiscussions)
	r.IsTemplate = boolDiff(gh.IsTemplate, d.IsTemplate)
	r.AllowSquashMerge = boolDiff(gh.AllowSquashMerge, d.AllowSquashMerge)
	r.AllowMergeCommit = boolDiff(gh.AllowMergeCommit, d.AllowMergeCommit)
	r.AllowRebaseMerge = boolDiff(gh.AllowRebaseMerge, d.AllowRebaseMerge)
	r.AllowAutoMerge = boolDiff(gh.AllowAutoMerge, d.AllowAutoMerge)
	r.DeleteBranchOnMerge = boolDiff(gh.DeleteBranchOnMerge, d.DeleteBranchOnMerge)
	if gh.Archived {
		r.Archived = ptr(true)
	}
	return r
}

func ptr[T any](v T) *T { return &v }

func boolDiff(actual, def bool) *bool {
	if actual != def {
		return &actual
	}
	return nil
}

// kv is one rendered "key: value" override line.
type kv struct {
	key   string
	value string
}

// lines returns the present override fields, in declaration order, each with
// its value already formatted (quoted strings, bare strings, or bool literals).
func (r Repository) lines() []kv {
	specs := []struct {
		val func() (string, bool)
		key string
	}{
		{key: "description", val: quoted(r.Description)},
		{key: "homepage", val: quoted(r.Homepage)},
		{key: "visibility", val: visibilityVal(r.Visibility)},
		{key: "default_branch", val: bareStr(r.DefaultBranch)},
		{key: "has_issues", val: boolVal(r.HasIssues)},
		{key: "has_projects", val: boolVal(r.HasProjects)},
		{key: "has_wiki", val: boolVal(r.HasWiki)},
		{key: "has_discussions", val: boolVal(r.HasDiscussions)},
		{key: "is_template", val: boolVal(r.IsTemplate)},
		{key: "allow_squash_merge", val: boolVal(r.AllowSquashMerge)},
		{key: "allow_merge_commit", val: boolVal(r.AllowMergeCommit)},
		{key: "allow_rebase_merge", val: boolVal(r.AllowRebaseMerge)},
		{key: "allow_auto_merge", val: boolVal(r.AllowAutoMerge)},
		{key: "delete_branch_on_merge", val: boolVal(r.DeleteBranchOnMerge)},
		{key: "archived", val: boolVal(r.Archived)},
	}
	out := make([]kv, 0, len(specs))
	for _, s := range specs {
		if v, ok := s.val(); ok {
			out = append(out, kv{s.key, v})
		}
	}
	return out
}

func quoted(p *string) func() (string, bool) {
	return func() (string, bool) {
		if p == nil {
			return "", false
		}
		return `"` + *p + `"`, true
	}
}

func bareStr(p *string) func() (string, bool) {
	return func() (string, bool) {
		if p == nil {
			return "", false
		}
		return *p, true
	}
}

func visibilityVal(p *repo.Visibility) func() (string, bool) {
	return func() (string, bool) {
		if p == nil {
			return "", false
		}
		return string(*p), true
	}
}

func boolVal(p *bool) func() (string, bool) {
	return func() (string, bool) {
		if p == nil {
			return "", false
		}
		if *p {
			return "true", true
		}
		return "false", true
	}
}

// Render returns the exact repos/<name>.yml byte content for the override file.
func (f File) Render() string {
	out := fmt.Sprintf("# %s/%s — overrides from %s defaults\n\n", f.Owner, f.Name, f.Source)
	if f.IsFork {
		out += "_fork: true\n\n"
	}
	return out + renderRepository(f.Repository.lines())
}

func renderRepository(lines []kv) string {
	if len(lines) == 0 {
		return "repository: {}\n"
	}
	out := "repository:\n"
	for _, l := range lines {
		out += fmt.Sprintf("  %s: %s\n", l.key, l.value)
	}
	return out
}
