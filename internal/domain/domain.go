// Package domain holds the named scalar types that flow through tools.admin.
// Every function and method parameter that carries a domain concept uses one of
// these instead of a bare string/int/bool, so the type system documents intent
// and prevents accidental argument transposition.
package domain

// Owner is a GitHub user or organization login.
type Owner string

// RepoName is a repository's short name (no owner prefix).
type RepoName string

// AccountType is the GitHub account type returned by GET /users/{owner}.
type AccountType string

// AccountTypeOrganization is the value GitHub reports for organization accounts.
const AccountTypeOrganization AccountType = "Organization"

// CommentSource names whose defaults a repo override file is diffed against.
type CommentSource string

const (
	// CommentSourceOrg is used when the owner is an organization.
	CommentSourceOrg CommentSource = "org"
	// CommentSourceAccount is used when the owner is a user account.
	CommentSourceAccount CommentSource = "account"
)

// Visibility is a repository's public/private visibility.
type Visibility string

const (
	// VisibilityPublic is a public repository.
	VisibilityPublic Visibility = "public"
	// VisibilityPrivate is a private repository.
	VisibilityPrivate Visibility = "private"
)

// Days is a retention window expressed in whole days.
type Days int

// KeepCount is the minimum number of runs to retain per workflow.
type KeepCount int

// DryRun toggles delete-vs-report behavior for cleanup-runs.
type DryRun bool

// Branch is a git branch name.
type Branch string

// Base is a pull-request base branch name.
type Base string

// SettingsPath is the path to the settings directory (e.g. ".github").
type SettingsPath string

// CreatedBefore is the upper-bound date (YYYY-MM-DD) for run listing.
type CreatedBefore string

// RunID is a GitHub Actions workflow-run identifier.
type RunID int64

// IsFork marks a repository as a fork.
type IsFork bool
