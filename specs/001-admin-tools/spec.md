# Admin Tools — Capabilities Specification

**Created**: 2026-02-15 **Updated**: 2026-02-15 **Status**: Active

## User Scenarios & Testing

### User Story 1 — Snapshot Live Settings (Priority: P1)

An org admin runs the snapshot command to capture the current state of all GitHub repositories and produce per-repo override YAML files showing how each repo differs from the org/account defaults.

**Why this priority**: Core value proposition. Without this, the tool has no purpose.

**Independent Test**: Run `tools.admin snapshot --owner nicerobot` against a real or mocked GitHub API and verify the output YAML files match expected overrides.

**Acceptance Scenarios**:

1. **Given** an org with 5 repos and a settings.yml defining defaults, **When** `tools.admin snapshot --owner myorg` runs, **Then** 5 YAML files are written to `.github/repos/` with correct overrides.
2. **Given** a user account with repos, **When** snapshot runs, **Then** the comment header says "overrides from account defaults" (not "org defaults").
3. **Given** a repo that was deleted since the last snapshot (confirmed 404), **When** snapshot runs, **Then** the stale override file is removed.
4. **Given** a fork repo, **When** snapshot runs, **Then** the override file includes `_fork: true`.
5. **Given** a repo with no differences from defaults, **When** snapshot runs, **Then** the override file contains `repository: {}`.
6. **Given** a stale override file for a repo that exists but was not returned by `list_repos()`, **When** snapshot runs, **Then** it aborts with an error and no files are modified.
7. **Given** a verification API error during stale-file checking, **When** snapshot runs, **Then** it aborts with an error and no files are modified.

---

### User Story 2 — Create Snapshot PR (Priority: P1)

After a snapshot produces changes, the create-pr command packages those changes into a pull request for review.

**Why this priority**: Paired with snapshot; together they form the complete workflow.

**Independent Test**: Run `tools.admin create-pr` after staging changes and verify a PR is created (or skipped if no changes exist).

**Acceptance Scenarios**:

1. **Given** staged changes in `.github/repos/`, **When** `tools.admin create-pr` runs, **Then** a commit is created, the branch is force-pushed, and a PR is opened.
2. **Given** no staged changes, **When** create-pr runs, **Then** it prints "No changes to commit." and exits without creating a PR.
3. **Given** an open PR already exists for the branch, **When** create-pr runs, **Then** it pushes the new commit but does not create a duplicate PR.

---

### User Story 3 — GitHub Action Consumption (Priority: P1)

Other org admin repos consume tools.admin as `uses: nicerobot/tools.admin@v1` in their workflows, replacing shell scripts and `yq` dependencies.

**Why this priority**: The packaging as a reusable action is the deployment mechanism.

**Independent Test**: Create a workflow in a test repo using `nicerobot/tools.admin@v1` with both `snapshot` and `create-pr` commands and verify it runs successfully.

**Acceptance Scenarios**:

1. **Given** a workflow with `command: snapshot` and `owner: myorg`, **When** the action runs, **Then** `entrypoint.sh` routes to `tools.admin snapshot --owner myorg`.
2. **Given** a workflow with `command: create-pr`, **When** the action runs, **Then** `entrypoint.sh` routes to `tools.admin create-pr` with the specified branch and base.
3. **Given** an unknown command, **When** the action runs, **Then** it exits with error and a clear message.

---

### User Story 4 — Cleanup Old Workflow Runs (Priority: P2)

An org admin runs the cleanup-runs command to delete old GitHub Actions workflow runs across repos, keeping a configurable number of recent runs per workflow and supporting dry-run preview.

**Why this priority**: Operational hygiene. Accumulated workflow runs slow down the Actions UI and consume storage. Not blocking for core snapshot functionality.

**Independent Test**: Run `tools.admin cleanup-runs --owner nicerobot --dry-run` and verify it lists runs that would be deleted without actually deleting them.

**Acceptance Scenarios**:

1. **Given** a repo with 10 completed runs older than 30 days across 2 workflows, **When** `cleanup-runs --days 30 --keep 2` runs, **Then** 6 runs are deleted (keeping 2 newest per workflow).
2. **Given** `--dry-run` is specified, **When** cleanup-runs executes, **Then** it prints what would be deleted but makes no API delete calls.
3. **Given** `--repo myrepo` is specified, **When** cleanup-runs executes, **Then** only that repo's runs are processed (no `list_repos()` call).
4. **Given** `--repo` is omitted, **When** cleanup-runs executes, **Then** all repos from `list_repos()` are processed.
5. **Given** a repo with no completed runs older than the cutoff, **When** cleanup-runs executes, **Then** nothing is deleted for that repo.
6. **Given** a repo with fewer runs than `--keep`, **When** cleanup-runs executes, **Then** no runs are deleted for that repo.

---

### Edge Cases

- What happens when `GH_TOKEN` is not set? Raises `RuntimeError` immediately.
- What happens when `settings.yml` is missing? Raises `FileNotFoundError`.
- What happens when GitHub API returns paginated results? Link-header pagination follows all pages.
- What happens when a repo has a `description` with special YAML characters? The value is double-quoted.
- What happens when the GitHub API returns extra fields not in the model? Pydantic silently ignores them.
- What happens when `list_repos()` returns fewer repos than expected (token scope)? Snapshot verifies each stale candidate individually and aborts if any still exist.
- What happens when verification of a stale repo returns a 500? Snapshot aborts — no files modified.
- What happens when `cleanup-runs` is run with `--keep 0`? All old runs are deleted (no minimum retained).
- What happens when `cleanup-runs` encounters a repo with no workflow runs? Skips silently.

## Functional Requirements

| ID | Name | Description | Acceptance |
| --- | --- | --- | --- |
| FR-001 | Fetch all repos | Fetch all repositories for a given GitHub user or org via the API | All repos returned including paginated results |
| FR-002 | User/org detection | Distinguish user accounts from orgs, use correct API endpoint | `type=owner` for users, `type=all` for orgs |
| FR-003 | Pagination | Handle GitHub API pagination via Link headers | All pages followed until no `rel="next"` |
| FR-004 | Load defaults | Load org/account defaults from `.github/settings.yml` | OrgSettings model populated correctly |
| FR-005 | Compute overrides | Compute per-repo overrides by comparing live settings against defaults | Only differing fields present in output |
| FR-006 | Write YAML | Write override YAML files in the exact settings-sync format | Matches format fidelity rules (Principle II) |
| FR-007 | Stale file cleanup | Remove overrides for deleted repos after individual 404 verification | Each candidate verified via GET before removal |
| FR-007a | Abort on scope mismatch | Abort entire snapshot if any stale candidate returns 200 | Zero files modified on abort |
| FR-008 | Create PR | Create git branch, commit changes, force push, open PR | PR created with correct branch and title |
| FR-009 | Skip empty PR | Skip PR creation when no changes are staged | "No changes to commit." message |
| FR-010 | Skip duplicate PR | Skip PR creation when open PR exists for the branch | No duplicate PR created |
| FR-011 | Schema validation | Validate generated YAML against the settings-sync JSON schema in tests | Tests pass against pinned v2.1.18 schema |
| FR-012 | Cleanup runs | Delete old workflow runs across repos via `cleanup-runs` command | Runs older than cutoff deleted |
| FR-013 | Group by workflow | Group runs by `workflow_id`, retain `--keep` newest per workflow | Correct number retained per workflow |
| FR-014 | Dry-run mode | `--dry-run` prints planned deletions without executing | No API delete calls in dry-run |
| FR-015 | Single/all repo mode | `--repo` for single repo, omit for all repos | Correct scoping of operation |

### Key Entities

- **OrgSettings**: The loaded settings.yml with repository defaults, labels, and collaborators.
- **GitHubRepository**: A repository as returned by the GitHub API, with computed visibility.
- **RepositoryOverrides**: The diff between a repo's live settings and the org defaults.
- **RepoOverrideFile**: A complete override file with metadata (owner, name, source, fork flag).
- **WorkflowRun**: A GitHub Actions workflow run with id, name, status, conclusion, created_at, and workflow_id.

## Success Criteria

| ID | Criterion |
| --- | --- |
| SC-001 | All 159+ unit tests pass |
| SC-002 | `mypy --strict` passes with zero errors |
| SC-003 | `ruff check` passes with zero violations |
| SC-004 | Generated YAML for real repos is byte-identical to existing overrides in `nicerobot/admin` |
| SC-005 | Docker image builds and runs all commands |
| SC-006 | Consumer workflows no longer need `Install yq` steps or shell scripts |
| SC-007 | Snapshot never deletes overrides when API token has insufficient scope |
| SC-008 | `cleanup-runs --dry-run` lists runs without making delete API calls |

## Integrations

- **GitHub REST API** (v2022-11-28): Repository listing, user/org detection, repo existence verification, workflow run listing and deletion.
- **Retired settings-sync app**: Downstream consumer of generated YAML files. Schema pinned to v2.1.18.
- **gh CLI**: Used for PR creation and existence checking in the `create-pr` command.
- **git CLI**: Used for branch, stage, commit, push operations in the `create-pr` command.
