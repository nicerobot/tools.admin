// Package overrides computes a repository's settings diff against the org
// defaults and renders it to the repos/<name>.yml format the safe-settings
// snapshot consumes. The render order is fixed to the field declaration order
// so output is byte-for-byte stable across runs.
package overrides

import (
	"fmt"
	"strings"

	"github.com/nicerobot/tools.admin/internal/domain"
	"github.com/nicerobot/tools.admin/internal/githubmodel"
	"github.com/nicerobot/tools.admin/internal/settings"
)

// Repository is the set of override fields, each a pointer so "absent" is
// distinct from a zero value. Field order here is the emit order.
type Repository struct {
	Description         *string
	Homepage            *string
	Visibility          *domain.Visibility
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
	Owner      domain.Owner
	Name       domain.RepoName
	Source     domain.CommentSource
	IsFork     domain.IsFork
	Repository Repository
}

// Compute diffs repo against defaults, returning the override document. The
// description/homepage are included when non-empty; archived only when true;
// every other field only when it differs from the default.
func Compute(
	repo githubmodel.Repository,
	defaults settings.RepositoryDefaults,
	owner domain.Owner,
	source domain.CommentSource,
) File {
	r := applyBools(applyStrings(Repository{}, repo, defaults), repo, defaults)
	return File{
		Owner:      owner,
		Name:       domain.RepoName(repo.Name),
		Source:     source,
		IsFork:     domain.IsFork(repo.Fork),
		Repository: r,
	}
}

func applyStrings(r Repository, repo githubmodel.Repository, d settings.RepositoryDefaults) Repository {
	if repo.Description != "" {
		r.Description = ptr(repo.Description)
	}
	if repo.Homepage != "" {
		r.Homepage = ptr(repo.Homepage)
	}
	if repo.Visibility() != d.Visibility {
		r.Visibility = ptr(repo.Visibility())
	}
	if repo.DefaultBranch != d.DefaultBranch {
		r.DefaultBranch = ptr(repo.DefaultBranch)
	}
	return r
}

func applyBools(r Repository, repo githubmodel.Repository, d settings.RepositoryDefaults) Repository {
	r.HasIssues = boolDiff(repo.HasIssues, d.HasIssues)
	r.HasProjects = boolDiff(repo.HasProjects, d.HasProjects)
	r.HasWiki = boolDiff(repo.HasWiki, d.HasWiki)
	r.HasDiscussions = boolDiff(repo.HasDiscussions, d.HasDiscussions)
	r.IsTemplate = boolDiff(repo.IsTemplate, d.IsTemplate)
	r.AllowSquashMerge = boolDiff(repo.AllowSquashMerge, d.AllowSquashMerge)
	r.AllowMergeCommit = boolDiff(repo.AllowMergeCommit, d.AllowMergeCommit)
	r.AllowRebaseMerge = boolDiff(repo.AllowRebaseMerge, d.AllowRebaseMerge)
	r.AllowAutoMerge = boolDiff(repo.AllowAutoMerge, d.AllowAutoMerge)
	r.DeleteBranchOnMerge = boolDiff(repo.DeleteBranchOnMerge, d.DeleteBranchOnMerge)
	if repo.Archived {
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
		key string
		val func() (string, bool)
	}{
		{"description", quoted(r.Description)},
		{"homepage", quoted(r.Homepage)},
		{"visibility", visibilityVal(r.Visibility)},
		{"default_branch", bareStr(r.DefaultBranch)},
		{"has_issues", boolVal(r.HasIssues)},
		{"has_projects", boolVal(r.HasProjects)},
		{"has_wiki", boolVal(r.HasWiki)},
		{"has_discussions", boolVal(r.HasDiscussions)},
		{"is_template", boolVal(r.IsTemplate)},
		{"allow_squash_merge", boolVal(r.AllowSquashMerge)},
		{"allow_merge_commit", boolVal(r.AllowMergeCommit)},
		{"allow_rebase_merge", boolVal(r.AllowRebaseMerge)},
		{"allow_auto_merge", boolVal(r.AllowAutoMerge)},
		{"delete_branch_on_merge", boolVal(r.DeleteBranchOnMerge)},
		{"archived", boolVal(r.Archived)},
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

func visibilityVal(p *domain.Visibility) func() (string, bool) {
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
	var b strings.Builder
	fmt.Fprintf(&b, "# %s/%s — overrides from %s defaults\n\n", f.Owner, f.Name, f.Source)
	if f.IsFork {
		b.WriteString("_fork: true\n\n")
	}
	renderRepository(&b, f.Repository.lines())
	return b.String()
}

func renderRepository(b *strings.Builder, lines []kv) {
	if len(lines) == 0 {
		b.WriteString("repository: {}\n")
		return
	}
	b.WriteString("repository:\n")
	for _, l := range lines {
		fmt.Fprintf(b, "  %s: %s\n", l.key, l.value)
	}
}
