# Admin Tools Constitution

## Core Principles

### I. Safe-Settings Compatibility

All generated YAML output must be valid input for github/safe-settings. Override files are validated against the pinned safe-settings JSON schema (v2.1.18) in tests. Fields not in the schema (e.g., `has_discussions`) are documented as intentional extensions and tested separately.

- Severity: critical
- Enforced: test suite (`test_schema_validation.py`)
- Source: project requirement; safe-settings is the downstream consumer

### II. Format Fidelity

Generated override YAML files must match the exact format of existing files in `nicerobot/admin` repo:

- Comment header: `# {owner}/{name} — overrides from {source} defaults`
- `_fork: true` line for forks (not a safe-settings directive; preserved for workflow compatibility)
- `description` and `homepage` values double-quoted; all other strings bare
- Booleans as bare `true`/`false`
- Empty overrides: `repository: {}`
- No trailing `---`

- Severity: critical
- Enforced: test suite (`test_settings_io.py`)
- Source: backward compatibility with existing admin repo workflows

### III. Complete Repository Enumeration

The GitHub API client must return all repositories owned by the account. The previous shell script used `/users/{OWNER}/repos` without `type=owner`, which silently omitted repos and caused PR #3 to delete valid override files.

- Users: `/users/{owner}/repos?type=owner&per_page=100` with Link-header pagination
- Orgs: `/orgs/{owner}/repos?type=all&per_page=100` with Link-header pagination

- Severity: critical
- Enforced: test suite (`test_github_api.py`)
- Source: incident — PR #3 in nicerobot/admin

### III-A. Verify-Then-Act for Destructive Operations

Any command that deletes files or resources must verify before acting. The snapshot command must individually confirm each stale override file candidate via `GET /repos/{owner}/{name}` before deleting it. If a repo exists but was not returned by `list_repos()` (indicating token scope mismatch), the entire operation aborts with zero files modified.

This pattern prevents data loss when:
- The API token lacks access to all repos (PR #4 in nicerobot/admin)
- Pagination fails silently
- Rate limiting truncates results

The two-phase flow: (1) gather and verify all changes, (2) apply changes only after all verification passes. No file is written or deleted until every stale candidate is confirmed as 404.

- Severity: critical
- Enforced: test suite (`test_snapshot.py` — abort tests, no-modification tests)
- Source: incident — PR #4 in nicerobot/admin; a scoped GitHub App token returned 3 of 62 repos, causing 59 override files to be proposed for deletion

### IV. Layer Separation

Code is organized into strict layers with unidirectional dependencies:

```
models/ (Pydantic data classes, no side effects)
  ↑
util/ (pure functions, no I/O)
  ↑
services/ (side-effect wrappers: HTTP, filesystem, subprocess)
  ↑
commands/ (orchestrators composing services and utils)
  ↑
cli.py (argparse entry point)
```

No layer may import from a layer above it.

- Severity: high
- Enforced: code review, import structure
- Source: architecture decision for testability and clarity

### V. Test-First, Full Coverage

Every function, model, and code path must have unit tests. The test suite is a CI gate — lint (ruff), typecheck (mypy --strict), and test (pytest) must all pass before merge.

- Severity: high
- Enforced: CI workflow (`.github/workflows/ci.yml`)
- Source: project requirement; replaces fragile shell scripts that had no tests

### VI. No TOML Unless Mandated

Per global convention, TOML is never used when an alternative format exists. `pyproject.toml` is the sole exception (PEP 517/518/621 mandates it). Tool configs use native formats: `ruff.toml` (ruff only supports TOML), `mypy.ini`, `pytest.ini`.

- Severity: medium
- Enforced: code review
- Source: global CLAUDE.md convention

## Security Constraints

- `GH_TOKEN` is never logged, committed, or passed as a command-line argument visible in process lists
- The token is read from the `GH_TOKEN` environment variable only
- Docker image uses multi-stage build to exclude source and dev dependencies from runtime

## Operational Rules

- The tool runs as a GitHub Action (Docker-based) or as a standalone CLI
- `snapshot` is the only command that reads repo settings from the GitHub API
- `cleanup-runs` is the only command that deletes workflow runs via the GitHub API
- `create-pr` is the only command that modifies git state
- `entrypoint.sh` routes action inputs to CLI subcommands; it contains no business logic
- Destructive commands (`cleanup-runs`) require explicit `--dry-run` to preview before acting

## Governance

Constitution supersedes all other practices. Amendments require documentation, developer approval, and updated tests.

**Version**: 1.1.0 | **Ratified**: 2026-02-15 | **Amended**: 2026-02-15 (Principle III-A, cleanup-runs operational rule)
