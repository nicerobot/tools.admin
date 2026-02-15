# Changelog

All notable changes to this project are documented here.

## [Unreleased]

### Added
- [Features in progress]

---

## [0.2.0] - 2026-02-15

### Added
- **Cleanup Runs Command** - Deletes old GitHub Actions workflow runs with configurable retention
  - Entry point: `src/admin_tools/commands/cleanup_runs.py:run_cleanup_runs`
  - CLI: `admin-tools cleanup-runs --owner OWNER [--repo REPO] [--days 30] [--keep 5] [--dry-run]`
  - Decision: Added as a new command in admin-tools (reuses existing API client, Docker infra) rather than a separate tool
- **WorkflowRun Model** - Pydantic model for GitHub Actions workflow run data
  - Entry point: `src/admin_tools/models/github.py:WorkflowRun`
- **Workflow Run API Methods** - `list_workflow_runs()` and `delete_workflow_run()` on GitHubClient
  - Entry point: `src/admin_tools/services/github_api.py`
  - Decision: Extended `_paginate()` with `items_key` param for wrapped API responses (GitHub returns `{ workflow_runs: [...] }`)
- **GitHub Action Inputs** - `cleanup-repo`, `cleanup-days`, `cleanup-keep`, `cleanup-dry-run` in action.yml
- **20 New Tests** - Covering cleanup-runs command, workflow run API methods, and CLI args

### Fixed
- **Snapshot Safety: Verify-Then-Act** - Snapshot now verifies each stale override file candidate individually via `GET /repos/{owner}/{name}` before any writes or deletes
  - Root cause: PR #4 in nicerobot/admin — a scoped GitHub App token returned 3 of 62 repos, causing 59 override files to be proposed for deletion
  - Prevention: If any stale candidate still exists (200), the entire operation aborts with zero files modified. Clear error message identifies token scope as the issue.
  - Entry point: `src/admin_tools/commands/snapshot.py:_run_snapshot_async`
- **repo_exists() API Method** - New method on GitHubClient for individual repo verification
  - Entry point: `src/admin_tools/services/github_api.py:GitHubClient.repo_exists`

### Decision Traces

#### Verify-Then-Act for Stale File Deletion
**Context**: PR #4 proposed deleting 59 valid override files because the API token only had access to 3 repos. This is the same class of bug as the original shell script PR #3.
**Options**: (1) Trust `list_repos()` results, (2) Threshold heuristic (abort if >50% stale), (3) Verify each stale candidate individually
**Selected**: Individual verification. One `GET /repos/{owner}/{name}` per stale candidate. 404 = confirmed gone, 200 = abort. Cheap check (one API call per stale file) with strong safety guarantee.
**Tradeoffs**: Slightly more API calls; prevents catastrophic data loss from token scope mismatches.
**Related**: constitution.md (Principle III-A), spec.md (FR-007a)

#### Cleanup Runs as Admin-Tools Command
**Context**: Need to bulk-delete old workflow runs. Could be a separate tool or a new command.
**Options**: (1) Shell script with `gh run delete`, (2) Separate standalone tool, (3) New command in admin-tools
**Selected**: New command in admin-tools. Reuses existing GitHubClient, Docker image, action.yml infrastructure. Single image for all admin operations.
**Tradeoffs**: Slightly larger scope for admin-tools; avoids duplicating API client, auth, and packaging.
**Related**: spec.md (User Story 4, FR-012 through FR-015)

#### Extended _paginate() with items_key
**Context**: GitHub workflow runs endpoint returns `{ workflow_runs: [...] }`, not a flat list. Existing `_paginate()` assumed flat lists.
**Options**: (1) Separate pagination method, (2) Post-process responses, (3) Optional `items_key` parameter
**Selected**: Optional `items_key` on existing `_paginate()`. Identical pagination logic — only item extraction differs. Existing callers unaffected.
**Tradeoffs**: Slightly more complex interface; reusable for any wrapped-response GitHub API endpoint.
**Related**: plan.md (services/github_api.py)

---

## [0.1.0] - 2026-02-15

### Added
- **Snapshot Command** - Fetches all repos for a GitHub user/org and writes per-repo override YAML files
  - Entry point: `src/admin_tools/commands/snapshot.py:run_snapshot`
  - Decision: Uses httpx async client instead of `gh api` CLI for type safety and testability
- **Create-PR Command** - Packages snapshot changes into a pull request
  - Entry point: `src/admin_tools/commands/create_pr.py:run_create_pr`
  - Decision: Uses `gh pr create` CLI for PR creation (simpler than API for this specific operation)
- **Docker-based GitHub Action** - Packaged as `nicerobot/admin-tools@v1` for consumption by org admin repos
  - Entry point: `entrypoint.sh` routes `INPUT_COMMAND` to CLI subcommands
  - Decision: Docker over JavaScript action for full control over runtime (Python, git, gh CLI)
- **Pydantic v2 Models** - Typed data models for settings, GitHub API responses, and overrides
  - Entry point: `src/admin_tools/models/`
  - Decision: Pydantic v2 over dataclasses for validation, serialization, and computed fields
- **Safe-settings Schema Validation** - Pinned JSON schema (v2.1.18) validates output in tests
  - Entry point: `tests/test_schema_validation.py`
  - Decision: Schema validation in tests over runtime validation; pinned version for stability
- **Complete Test Suite** - 139+ unit tests covering every function and code path
  - Entry point: `tests/`
  - Decision: httpx MockTransport for API tests; subprocess mocking for git tests
- **CI Pipeline** - Lint (ruff), typecheck (mypy --strict), test (pytest), Docker build on tags
  - Entry point: `.github/workflows/ci.yml`

### Fixed
- **Complete Repository Enumeration** - Fixed bug where `/users/{OWNER}/repos` without `type=owner` omitted repos, causing PR #3 in nicerobot/admin to propose deleting valid override files
  - Root cause: GitHub API default behavior for user repos endpoint
  - Prevention: Uses `type=owner` for users and `type=all` for orgs; tested in `test_github_api.py`

### Decision Traces

#### Replace Shell Scripts with Python
**Context**: `nicerobot/admin` used bash scripts with `jq`, `yq`, and `gh` CLI in fragile pipelines. The same logic needed replication across multiple org admin repos.
**Options**: (1) Fix shell scripts, (2) Rewrite in Python as reusable action, (3) Use an existing tool
**Selected**: Python rewrite as Docker-based GitHub Action. Shell scripts had no tests, no type checking, and the `jq`/`yq` pipeline was brittle. No existing tool matched the safe-settings override file format.
**Tradeoffs**: More initial effort; gains testability, type safety, and reusability across orgs.
**Related**: constitution.md (Layer Separation, Test-First), plan.md (Architecture Decisions)

#### Hand-Formatted YAML Output
**Context**: Generated YAML must exactly match existing override files (comment headers, selective quoting, fork flags).
**Options**: (1) PyYAML dumper with custom representers, (2) Hand-formatted string output, (3) Jinja2 templates
**Selected**: Hand-formatted strings. PyYAML's default quoting and ordering don't match the required format. Custom representers would fight the library. Jinja2 adds a dependency for a simple template.
**Tradeoffs**: More explicit formatting code; gains exact control over output format.
**Related**: constitution.md (Format Fidelity), spec.md (FR-006)

#### Async httpx for GitHub API
**Context**: Need to fetch repos with pagination. Shell scripts used `gh api --paginate` piped through `jq`.
**Options**: (1) `gh api` subprocess calls, (2) `requests` synchronous, (3) `httpx` async
**Selected**: httpx async. Allows concurrent operations in future. MockTransport enables clean testing without mocking. Async context manager pattern fits well.
**Tradeoffs**: Slightly more complex setup; gains testability and future concurrency.
**Related**: plan.md (services/github_api.py)
