package cleanupruns

import "github.com/nicerobot/tools.admin/internal/repo"

// Config holds the flags for the cleanup-runs command. An empty Owner triggers
// GITHUB_REPOSITORY auto-detection; an empty Repo scans all repos under the
// owner. Its fields are bound by the CLI tier and read by Run; it carries no
// behavior. Every field reuses the implementation tier's named types, so no
// domain-local types.go is needed.
type Config struct {
	Owner    repo.Owner
	Repo     repo.Name
	Days     repo.Days
	Keep     repo.KeepCount
	IsDryRun repo.DryRun
}
