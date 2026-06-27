// Package githubmodel holds the JSON-decoded shapes returned by the GitHub REST
// API that tools.admin consumes. The fields are bare types because they are
// populated by encoding/json (a decoder we do not control); the domain-typed
// surface is exposed through the accessor methods and the calling packages.
package githubmodel

import "github.com/nicerobot/tools.admin/internal/domain"

// User is the subset of GET /users/{owner} that drives account-type detection.
type User struct {
	Login string `json:"login"`
	Type  string `json:"type"`
}

// AccountType returns the user's account type as a domain value.
func (u User) AccountType() domain.AccountType { return domain.AccountType(u.Type) }

// Repository is the subset of a GitHub repository object that snapshot diffs
// against the org/account defaults.
type Repository struct {
	Name                string `json:"name"`
	Description         string `json:"description"`
	Homepage            string `json:"homepage"`
	Private             bool   `json:"private"`
	DefaultBranch       string `json:"default_branch"`
	HasIssues           bool   `json:"has_issues"`
	HasProjects         bool   `json:"has_projects"`
	HasWiki             bool   `json:"has_wiki"`
	HasDiscussions      bool   `json:"has_discussions"`
	IsTemplate          bool   `json:"is_template"`
	AllowSquashMerge    bool   `json:"allow_squash_merge"`
	AllowMergeCommit    bool   `json:"allow_merge_commit"`
	AllowRebaseMerge    bool   `json:"allow_rebase_merge"`
	AllowAutoMerge      bool   `json:"allow_auto_merge"`
	DeleteBranchOnMerge bool   `json:"delete_branch_on_merge"`
	Archived            bool   `json:"archived"`
	Fork                bool   `json:"fork"`
}

// Visibility derives the public/private visibility from the private flag.
func (r Repository) Visibility() domain.Visibility {
	if r.Private {
		return domain.VisibilityPrivate
	}
	return domain.VisibilityPublic
}

// WorkflowRun is the subset of a GitHub Actions run object that cleanup-runs
// groups, sorts, and prunes.
type WorkflowRun struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	CreatedAt  string `json:"created_at"`
	WorkflowID int64  `json:"workflow_id"`
}
