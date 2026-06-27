// Package adminerr holds the sentinel-error type for tools.admin. Every error
// the program can emit is a const of type Error, so each path is matchable with
// errors.Is instead of by string comparison. It mirrors the Error types in the
// sibling tooling repos so error construction stays uniform across the ecosystem.
package adminerr

import (
	"fmt"
	"strings"
)

// Error is the package sentinel-error type.
type Error string

// Error implements the error interface.
func (e Error) Error() string { return string(e) }

// With wraps a cause and appends contextual args. A non-nil cause is joined
// with %w so errors.Is still matches both the sentinel and the cause. The args
// render space-separated, so callers pass clean key/value pairs —
// .With(err, "owner", owner) — without baking separators into the key.
func (e Error) With(err error, args ...any) error {
	out := error(e)
	if err != nil {
		out = fmt.Errorf("%w: %w", e, err)
	}
	if len(args) > 0 {
		out = fmt.Errorf("%w: %s", out, strings.TrimSuffix(fmt.Sprintln(args...), "\n"))
	}
	return out
}

// Sentinels emitted across tools.admin.
const (
	// ErrNoToken means the GH_TOKEN environment variable is unset.
	ErrNoToken Error = "GH_TOKEN environment variable must be set"
	// ErrSettingsNotFound means the org settings.yml file is absent.
	ErrSettingsNotFound Error = "settings file not found"
	// ErrInvalidSettings means the org settings.yml file failed to parse.
	ErrInvalidSettings Error = "settings file is not valid YAML"
	// ErrStaleRepoExists means a stale-candidate repo still exists, signalling
	// the token lacks access to all repos; snapshot aborts before any write.
	ErrStaleRepoExists Error = "repo exists but was not returned by list_repos; aborting to prevent data loss"
	// ErrNoTarget means cleanup-runs has no owner and GITHUB_REPOSITORY is unset.
	ErrNoTarget Error = "--owner is required when GITHUB_REPOSITORY is unset"
	// ErrHTTPStatus means a GitHub API call returned an unexpected status.
	ErrHTTPStatus Error = "unexpected GitHub API status"
	// ErrDecodeResponse means a GitHub API response body failed to decode.
	ErrDecodeResponse Error = "failed to decode GitHub API response"
	// ErrCommand means an external git/gh command failed.
	ErrCommand Error = "command failed"
	// ErrWriteFile means a repos/<name>.yml file could not be written.
	ErrWriteFile Error = "failed to write override file"
	// ErrListRepoFiles means the repos directory could not be listed.
	ErrListRepoFiles Error = "failed to list repo files"
	// ErrRemoveFile means a stale override file could not be removed.
	ErrRemoveFile Error = "failed to remove override file"
)
