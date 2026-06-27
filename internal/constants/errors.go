package constants

// Keep these constants sorted alphabetically.
const (
	// ErrCommand means an external git/gh command failed.
	ErrCommand Error = "command failed"
	// ErrDecodeResponse means a GitHub API response body failed to decode.
	ErrDecodeResponse Error = "failed to decode GitHub API response"
	// ErrHTTPStatus means a GitHub API call returned an unexpected status.
	ErrHTTPStatus Error = "unexpected GitHub API status"
	// ErrInvalidSettings means the org settings.yml file failed to parse.
	ErrInvalidSettings Error = "settings file is not valid YAML"
	// ErrInvalidValue means a value (e.g. an output format) is not recognized.
	ErrInvalidValue Error = "invalid value"
	// ErrListRepoFiles means the repos directory could not be listed.
	ErrListRepoFiles Error = "failed to list repo files"
	// ErrNoAuth means the GH_TOKEN environment variable is unset.
	ErrNoAuth Error = "GH_TOKEN environment variable must be set"
	// ErrNoTarget means cleanup-runs has no owner and GITHUB_REPOSITORY is unset.
	ErrNoTarget Error = "--owner is required when GITHUB_REPOSITORY is unset"
	// ErrRemoveFile means a stale override file could not be removed.
	ErrRemoveFile Error = "failed to remove override file"
	// ErrSettingsNotFound means the org settings.yml file is absent.
	ErrSettingsNotFound Error = "settings file not found"
	// ErrStaleRepoExists means a stale-candidate repo still exists, signalling
	// the token lacks access to all repos; snapshot aborts before any write.
	ErrStaleRepoExists Error = "repo exists but was not returned by list_repos; aborting to prevent data loss"
	// ErrWriteFile means a repos/<name>.yml file could not be written.
	ErrWriteFile Error = "failed to write override file"
)
