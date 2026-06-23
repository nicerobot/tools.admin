// Package cleanupruns orchestrates the cleanup-runs command: resolve the target
// owner/repo (explicit or from GITHUB_REPOSITORY), then per repo list completed
// runs older than the cutoff, group by workflow, keep the newest N, and delete
// (or, in dry-run, count) the rest.
//
// It defines the command's Config (the flags the CLI binds) and Run (the
// orchestration entry point). Run delegates all GitHub I/O to the reusable
// internal/github package and holds no CLI or output-formatting logic. This is
// the domain tier between the app tier (internal/app/commands/cleanupruns) and
// the implementation tier (internal/github).
package cleanupruns
