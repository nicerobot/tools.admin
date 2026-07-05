// Package overrides computes a repository's settings diff against the org
// defaults and renders it to the repos/<name>.yml format the settings
// snapshot consumes. The render order is fixed to the field declaration order
// so output is byte-for-byte stable across runs.
package overrides

import (
	"fmt"
	"strconv"

	"github.com/nicerobot/tools.admin/internal/github"
	"github.com/nicerobot/tools.admin/internal/repo"
	"github.com/nicerobot/tools.admin/internal/settings"
)

// Repository is the set of override fields, each a pointer so "absent" is
// distinct from a zero value. Field order here is the emit order.
type Repository struct {
	Description               *string
	Homepage                  *string
	Visibility                *repo.Visibility
	DefaultBranch             *string
	HasIssues                 *bool
	HasProjects               *bool
	HasWiki                   *bool
	HasDiscussions            *bool
	IsTemplate                *bool
	CanSquashMerge            *bool
	CanMergeCommit            *bool
	CanRebaseMerge            *bool
	CanAutoMerge              *bool
	ShouldDeleteBranchOnMerge *bool
	IsArchived                *bool
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
		IsFork:     repo.IsFork(gh.IsFork),
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
	r.CanSquashMerge = boolDiff(gh.CanSquashMerge, d.CanSquashMerge)
	r.CanMergeCommit = boolDiff(gh.CanMergeCommit, d.CanMergeCommit)
	r.CanRebaseMerge = boolDiff(gh.CanRebaseMerge, d.CanRebaseMerge)
	r.CanAutoMerge = boolDiff(gh.CanAutoMerge, d.CanAutoMerge)
	r.ShouldDeleteBranchOnMerge = boolDiff(gh.ShouldDeleteBranchOnMerge, d.ShouldDeleteBranchOnMerge)
	if gh.IsArchived {
		r.IsArchived = ptr(true)
	}
	return r
}

func ptr[T any](v T) *T { return &v }

// boolDiff returns a pointer to the live flag when it differs from the org
// default, nil when it matches. It is generic over any bool-shaped flag so the
// mirror structs' fields pass directly, whatever named flag type they use.
func boolDiff[T ~bool](isLive, isDefault T) *bool {
	if isLive != isDefault {
		v := bool(isLive)
		return &v
	}
	return nil
}

// kv is one rendered "key: value" override line.
type kv struct {
	key   string
	value string
}

// lineSpec pairs an override key with the producer of its optional rendered value.
type lineSpec struct {
	val func() (string, bool)
	key string
}

// lines returns the present override fields, in declaration order, each with
// its value already formatted (quoted strings, bare strings, or bool literals).
func (r Repository) lines() []kv {
	quote := func(s string) string { return `"` + s + `"` }
	bare := func(s string) string { return s }
	vis := func(v repo.Visibility) string { return string(v) }
	specs := []lineSpec{
		{key: "description", val: optVal(r.Description, quote)},
		{key: "homepage", val: optVal(r.Homepage, quote)},
		{key: "visibility", val: optVal(r.Visibility, vis)},
		{key: "default_branch", val: optVal(r.DefaultBranch, bare)},
		{key: "has_issues", val: optVal(r.HasIssues, strconv.FormatBool)},
		{key: "has_projects", val: optVal(r.HasProjects, strconv.FormatBool)},
		{key: "has_wiki", val: optVal(r.HasWiki, strconv.FormatBool)},
		{key: "has_discussions", val: optVal(r.HasDiscussions, strconv.FormatBool)},
		{key: "is_template", val: optVal(r.IsTemplate, strconv.FormatBool)},
		{key: "allow_squash_merge", val: optVal(r.CanSquashMerge, strconv.FormatBool)},
		{key: "allow_merge_commit", val: optVal(r.CanMergeCommit, strconv.FormatBool)},
		{key: "allow_rebase_merge", val: optVal(r.CanRebaseMerge, strconv.FormatBool)},
		{key: "allow_auto_merge", val: optVal(r.CanAutoMerge, strconv.FormatBool)},
		{key: "delete_branch_on_merge", val: optVal(r.ShouldDeleteBranchOnMerge, strconv.FormatBool)},
		{key: "archived", val: optVal(r.IsArchived, strconv.FormatBool)},
	}
	out := make([]kv, 0, len(specs))
	for _, s := range specs {
		if v, ok := s.val(); ok {
			out = append(out, kv{s.key, v})
		}
	}
	return out
}

// optVal renders an optional override field: absent when p is nil, otherwise
// render applied to the pointed-to value.
func optVal[T any](p *T, render func(T) string) func() (string, bool) {
	return func() (string, bool) {
		if p == nil {
			return "", false
		}
		return render(*p), true
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
