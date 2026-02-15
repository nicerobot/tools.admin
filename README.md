# admin-tools

[![CI](https://github.com/nicerobot/admin-tools/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/nicerobot/admin-tools/actions/workflows/ci.yml)
[![Release](https://github.com/nicerobot/admin-tools/actions/workflows/release.yml/badge.svg)](https://github.com/nicerobot/admin-tools/actions/workflows/release.yml)

GitHub Action for managing repository settings across organizations.

**This repo is intentionally public.** It is used as a shared action across multiple
orgs. Do not add org-specific configuration, secrets, or credentials to this repo.
Keep all content generic and reusable.

## Commands

| Command | Description |
|---|---|
| `snapshot` | Capture live GitHub repo settings and write them as YAML overrides |
| `create-pr` | Commit snapshot changes and open a PR |
| `cleanup-runs` | Delete old workflow runs across repos |

## Usage

```yaml
- uses: nicerobot/admin-tools@main
  with:
    command: snapshot
    owner: my-org
  env:
    GH_TOKEN: ${{ steps.app-token.outputs.token }}

- uses: nicerobot/admin-tools@main
  with:
    command: create-pr
  env:
    GH_TOKEN: ${{ steps.app-token.outputs.token }}
```

## Cleaning up old workflow runs

`cleanup-runs` deletes a repository's old completed runs (success and
failure), keeping the newest few per workflow. Run inside a repository's
Actions context it needs **no configuration** — it auto-detects the repo from
`GITHUB_REPOSITORY` and uses the run's `GITHUB_TOKEN` (which carries
`actions: write` for its own repo).

The reusable workflow [`cleanup-runs.yml`](.github/workflows/cleanup-runs.yml)
wraps this. To keep a repo's runs clean, drop
[`examples/cleanup-runs-weekly.yml`](examples/cleanup-runs-weekly.yml) into it
as `.github/workflows/cleanup-runs.yml`:

```yaml
name: Cleanup runs
on:
  schedule:
    - cron: "0 10 * * 0" # weekly, Sundays 10:00 UTC
  workflow_dispatch:
permissions:
  actions: write
jobs:
  cleanup:
    uses: nicerobot/admin-tools/.github/workflows/cleanup-runs.yml@main
```

Reusable workflow inputs: `days` (default `30`), `keep` (default `5`),
`dry-run` (default `false`). Optional secret `token` overrides the default
`GITHUB_TOKEN` (needed for cross-repo or org-wide cleanup).

## Inputs

| Input | Required | Default | Description |
|---|---|---|---|
| `command` | yes | | `snapshot`, `create-pr`, or `cleanup-runs` |
| `owner` | no | | GitHub user or organization (required for `snapshot`; `cleanup-runs` auto-detects the current repo from `GITHUB_REPOSITORY` when omitted) |
| `settings-path` | no | `.github` | Path to settings directory |
| `branch` | no | `safe-settings/snapshot` | Branch name for `create-pr` |
| `base` | no | `main` | Base branch for `create-pr` |
| `cleanup-repo` | no | | Single repo for `cleanup-runs` (omit for all repos) |
| `cleanup-days` | no | `30` | Delete runs older than N days |
| `cleanup-keep` | no | `5` | Keep at least N runs per workflow |
| `cleanup-dry-run` | no | `false` | Print what would be deleted without deleting |
