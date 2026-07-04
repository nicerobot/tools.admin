# Admin Tools — Architecture Plan

**Language/Version**: Python 3.12+
**Primary Dependencies**: httpx (async HTTP), pydantic v2 (validation), pyyaml (YAML parsing)
**Testing**: pytest, pytest-asyncio, jsonschema (schema validation)
**Target Platform**: Linux (Docker container), macOS (local development)
**Project Type**: Single CLI application packaged as a Docker-based GitHub Action

## System Overview

Admin Tools snapshots live GitHub repository settings and produces per-repo YAML override files compatible with the retired settings-sync app. It replaces fragile shell scripts (`snapshot-live-settings.sh`, `create-snapshot-pr.sh`) from `nicerobot/admin` with tested, typed Python code.

Consumer org admin repos invoke it as `uses: nicerobot/tools.admin@v1`.

## Directory Structure

```
tools.admin/
├── .github/workflows/ci.yml     # lint, typecheck, test, docker build+push
├── .specify/                     # Spec Kit artifacts
├── schema/                       # Pinned settings-sync JSON schema (v2.1.18)
├── src/admin_tools/
│   ├── __init__.py               # Package version
│   ├── __main__.py               # python -m admin_tools entry
│   ├── py.typed                  # PEP 561 marker
│   ├── cli.py                    # argparse: snapshot, create-pr, cleanup-runs subcommands
│   ├── commands/
│   │   ├── snapshot.py           # Orchestrator: load settings, fetch, verify, diff, write
│   │   ├── create_pr.py          # Orchestrator: git branch, stage, commit, push, PR
│   │   └── cleanup_runs.py       # Orchestrator: list runs, group, filter, delete
│   ├── models/
│   │   ├── settings.py           # OrgSettings, RepositoryDefaults, Label, Collaborator
│   │   ├── github.py             # GitHubRepository, GitHubUser, WorkflowRun
│   │   └── overrides.py          # RepositoryOverrides (all Optional), RepoOverrideFile
│   ├── services/
│   │   ├── github_api.py         # Async httpx client, Link-header pagination, repo verification, workflow run ops
│   │   ├── settings_io.py        # Load settings.yml, write override YAML, list files
│   │   └── git.py                # Thin subprocess wrappers for git/gh CLI
│   └── util/
│       └── diff.py               # Pure function: compute overrides from repo vs defaults
├── tests/                        # 159+ unit tests
│   ├── conftest.py               # Shared fixtures
│   ├── test_models.py            # Pydantic model validation
│   ├── test_diff.py              # Override computation logic
│   ├── test_settings_io.py       # YAML I/O format fidelity
│   ├── test_github_api.py        # httpx MockTransport, pagination, endpoints, repo_exists, workflow runs
│   ├── test_snapshot.py          # Full snapshot command orchestration + verify-then-act safety tests
│   ├── test_create_pr.py         # PR creation flow
│   ├── test_cleanup_runs.py      # Workflow run cleanup: delete, keep, dry-run, single/all repos
│   ├── test_cli.py               # Argparse routing for all subcommands
│   ├── test_git.py               # Subprocess wrappers
│   └── test_schema_validation.py # settings-sync JSON schema validation
├── Dockerfile                    # Multi-stage: python:3.12-slim + git + gh
├── entrypoint.sh                 # Routes action inputs to CLI
├── action.yml                    # GitHub Action metadata (docker type)
├── pyproject.toml                # PEP-mandated build config
├── ruff.toml                     # Linter config
├── mypy.ini                      # Type checker config (strict + pydantic plugin)
└── pytest.ini                    # Test runner config
```

## Layer Architecture

```
┌─────────────────────────────┐
│         cli.py              │  argparse → match/case dispatch
├─────────────────────────────┤
│       commands/             │  Orchestrators (snapshot, create-pr)
├─────────────────────────────┤
│       services/             │  Side effects (HTTP, filesystem, git subprocess)
├─────────────────────────────┤
│        util/                │  Pure functions (diff logic)
├─────────────────────────────┤
│       models/               │  Pydantic data classes (no side effects)
└─────────────────────────────┘
```

Dependencies flow downward only. Each layer is independently testable.

## Key Components

### models/settings.py
Defines `OrgSettings` loaded from `.github/settings.yml`. `RepositoryDefaults` carries the org/account default values that repos are compared against. `Label` and `Collaborator` capture the labels and collaborator sections.

### models/github.py
`GitHubRepository` models the GitHub API response with a `@computed_field` for `visibility` (derived from `private` boolean). Extra API fields are silently ignored (`model_config` with extra="ignore" via Pydantic default). `WorkflowRun` models a GitHub Actions workflow run with id, name, status, conclusion, created_at, and workflow_id for the `cleanup-runs` command.

### models/overrides.py
`RepositoryOverrides` has all-Optional fields so `model_dump(exclude_none=True)` yields only the fields that differ from defaults. `RepoOverrideFile` wraps overrides with metadata (owner, name, comment source, fork flag).

### util/diff.py
`compute_overrides()` is the core pure function. Compares a `GitHubRepository` against `RepositoryDefaults` field by field. `description` and `homepage` are always included when non-empty (they have no default). `archived` is only included when True. Returns a `RepoOverrideFile`.

### services/github_api.py
`GitHubClient` is an async context manager wrapping httpx. `_paginate()` follows `Link: <url>; rel="next"` headers and supports an `items_key` parameter for APIs that return wrapped results (e.g., `workflow_runs` key in the Actions API). `list_repos()` selects the correct endpoint: `/orgs/{owner}/repos?type=all` for organizations, `/users/{owner}/repos?type=owner` for users. `repo_exists()` verifies individual repos via `GET /repos/{owner}/{name}` (True on 200, False on 404, raises on other status). `list_workflow_runs()` and `delete_workflow_run()` support the cleanup-runs command.

### services/settings_io.py
`write_repo_override()` produces hand-formatted YAML matching the existing file format exactly. Only `description` and `homepage` are double-quoted; all other string values are bare. Booleans are bare `true`/`false`.

### services/git.py
Thin subprocess wrappers for git and gh CLI operations. The `create-pr` command uses these exclusively. No business logic lives here.

### commands/snapshot.py
`run_snapshot()` orchestrates a verify-then-act pattern: load settings, fetch repos via API, compute overrides, identify stale override files, verify each stale candidate individually via `repo_exists()`, then — only after all verification passes — write new files and delete confirmed-gone files. If any stale candidate still exists (indicating token scope mismatch), the entire operation aborts with zero files modified.

### commands/create_pr.py
`run_create_pr()` orchestrates: configure bot identity, checkout branch, stage repos directory, commit if changes exist, force push, create PR if none exists.

### commands/cleanup_runs.py
`run_cleanup_runs()` orchestrates: resolve target repos (single or all), fetch completed workflow runs older than the cutoff date, group by workflow_id, retain the newest `--keep` per workflow, delete the rest (or print in dry-run mode). Prints a summary of repos scanned, runs deleted, and runs kept.

## Data Flow

### Snapshot

```
settings.yml → OrgSettings → RepositoryDefaults
                                    ↓
GitHub API → [GitHubRepository] → compute_overrides() → [RepoOverrideFile]
                                                              ↓
                                              list_existing_repo_files() → stale candidates
                                                              ↓
                                          repo_exists() per stale → verify all 404
                                                              ↓
                                   [all verified] → write_repo_override() → delete confirmed-gone
                                   [any 200]      → ABORT (no files modified)
                                   [any error]    → ABORT (exception propagates)
```

### Cleanup Runs

```
--repo flag → single repo name   OR   list_repos(owner) → [repo names]
                                            ↓
            for each repo: list_workflow_runs(created_before=cutoff)
                                            ↓
                          group by workflow_id → sort newest first → skip keep
                                            ↓
                    dry_run? → print          OR   delete_workflow_run() per run
                                            ↓
                                       summary: scanned, deleted, kept
```

### Create PR

```
.github/repos/ → git add → git diff --cached → git commit → git push → gh pr create
```

## Architecture Decisions

### Python over shell
**Context**: Shell scripts using `jq`, `yq`, and `gh api` were fragile and untestable.
**Selected**: Python 3.12+ with httpx, pydantic, pyyaml.
**Tradeoffs**: Requires Docker for the GitHub Action; gains type safety, testing, and pagination handling.

### Hand-formatted YAML over PyYAML dumper
**Context**: Generated YAML must match existing file format exactly (comment headers, selective quoting, fork flags).
**Selected**: String concatenation with explicit formatting rules.
**Tradeoffs**: More code to maintain; gains exact format control without fighting YAML library defaults.

### httpx async over gh CLI
**Context**: `gh api --paginate` is convenient but parsing JSON output in shell is fragile.
**Selected**: httpx async client with Link-header pagination.
**Tradeoffs**: More setup; gains proper error handling, typed responses, and testability via MockTransport.

### argparse over Click/Typer
**Context**: Only two subcommands needed.
**Selected**: stdlib argparse.
**Tradeoffs**: More verbose; zero extra dependencies.

### Pinned settings-sync schema
**Context**: Generated YAML must be valid input for the retired settings-sync app.
**Selected**: Pinned JSON schema (v2.1.18) validated in tests.
**Tradeoffs**: Must manually update when the settings-sync schema changes; gains CI-enforced compatibility.

### Verify-then-act for stale file deletion
**Context**: PR #4 in nicerobot/admin proposed deleting 59 of 62 override files because a scoped GitHub App token only had access to 3 repos. `list_repos()` returned 3 repos and the stale file logic treated all other override files as deletable.
**Selected**: Two-phase approach — verify every stale candidate individually via `GET /repos/{owner}/{name}`, then apply changes only after all verifications pass.
**Alternatives considered**: (1) Trust `list_repos()` results — rejected, the PR #4 incident proves this is unsafe. (2) Compare counts (e.g., abort if more than 50% stale) — rejected, threshold-based heuristics fail at boundary conditions and don't diagnose the root cause.
**Tradeoffs**: One extra API call per stale candidate; gains defense-in-depth against token scope mismatches, pagination failures, and rate limiting. The abort message tells the user exactly what's wrong.

### items_key parameter for _paginate
**Context**: GitHub's workflow runs endpoint returns `{ workflow_runs: [...], total_count: N }`, not a flat list. The existing `_paginate()` method assumed flat list responses.
**Selected**: Added an optional `items_key` parameter to `_paginate()`. When set, items are extracted from `data[items_key]` instead of treating the response as a flat list.
**Alternatives considered**: (1) Separate pagination method — rejected, duplicates 95% of the logic. (2) Post-process responses — rejected, breaks Link-header following.
**Tradeoffs**: Slightly more complex `_paginate()` interface; gains reusability for any wrapped-response GitHub API endpoint.

## Quality Attributes

| ID | Attribute | Requirement |
|---|---|---|
| NFR-001 | Safety | Destructive operations must verify-then-act; abort on any verification failure with zero modifications |
| NFR-002 | Compatibility | Generated YAML must be byte-identical to existing `nicerobot/admin` override files |
| NFR-003 | Type safety | `mypy --strict` must pass with zero errors across the entire codebase |
| NFR-004 | Lint compliance | `ruff check` must pass with zero violations |
| NFR-005 | Test coverage | All 159+ unit tests must pass; every code path exercised |
| NFR-006 | Security | `GH_TOKEN` never logged, committed, or visible in process lists |
| NFR-007 | Portability | Runs as Docker-based GitHub Action and as standalone CLI on macOS/Linux |
| NFR-008 | Maintainability | Strict layer separation with unidirectional dependencies; each layer independently testable |
