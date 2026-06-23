package github

import "github.com/nicerobot/tools.admin/internal/repo"

// User is the subset of GET /users/{owner} that drives account-type detection.
type User struct {
	Login string `json:"login"`
	Type  string `json:"type"`
}

// AccountType returns the user's account type as a domain value.
func (u User) AccountType() repo.AccountType { return repo.AccountType(u.Type) }

// Repository is the subset of a GitHub repository object that snapshot diffs
// against the org/account defaults.
type Repository struct {
	DefaultBranch       string `json:"default_branch"`
	Description         string `json:"description"`
	Homepage            string `json:"homepage"`
	Name                string `json:"name"`
	HasDiscussions      bool   `json:"has_discussions"`
	HasIssues           bool   `json:"has_issues"`
	HasProjects         bool   `json:"has_projects"`
	HasWiki             bool   `json:"has_wiki"`
	Private             bool   `json:"private"`
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
func (r Repository) Visibility() repo.Visibility {
	if r.Private {
		return repo.VisibilityPrivate
	}
	return repo.VisibilityPublic
}

// WorkflowRun is the subset of a GitHub Actions run object that cleanup-runs
// groups, sorts, and prunes.
type WorkflowRun struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	CreatedAt  string `json:"created_at"`
	ID         int64  `json:"id"`
	WorkflowID int64  `json:"workflow_id"`
}
