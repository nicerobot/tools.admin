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

## Inputs

| Input | Required | Default | Description |
|---|---|---|---|
| `command` | yes | | `snapshot`, `create-pr`, or `cleanup-runs` |
| `owner` | no | | GitHub user or organization (required for `snapshot` and `cleanup-runs`) |
| `settings-path` | no | `.github` | Path to settings directory |
| `branch` | no | `safe-settings/snapshot` | Branch name for `create-pr` |
| `base` | no | `main` | Base branch for `create-pr` |
| `cleanup-repo` | no | | Single repo for `cleanup-runs` (omit for all repos) |
| `cleanup-days` | no | `30` | Delete runs older than N days |
| `cleanup-keep` | no | `5` | Keep at least N runs per workflow |
| `cleanup-dry-run` | no | `false` | Print what would be deleted without deleting |
