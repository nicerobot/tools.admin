# Port the `tools.admin` CLI from Python to Go

**Date:** 2026-06-27 · **Level:** Decomposed · **Owner:** rnix

## Goal

Replace the Python `admin_tools` package (the `tools.admin` console CLI: `snapshot`, `create-pr`, `cleanup-runs`) with a Go implementation that satisfies the home Go quality bar, **eliminating all TOML** (`pyproject.toml`, `ruff.toml`, `mypy.ini`, `pytest.ini`, `uv.lock`). The repo's shell/composite actions (`refresh-infra/`, `deploy-*/`, `go-tooling/`, `make/`) are unaffected — only the Python CLI is ported. `tools.repository` (already Go) is the reference.

**Why:** TOML is banned by policy ([global CLAUDE.md](../../../../.claude/CLAUDE.md)); Python packaging forces `pyproject.toml`, so the only way to remove it is to stop the CLI being Python. Go is the sanctioned CLI language.

## Behavioral contract (must be preserved exactly)

### `snapshot --owner <o> --settings-path <.github>`
- Load `<settings-path>/settings.yml` → org defaults (`repository:` block); missing file is a fatal error.
- Resolve account type via `GET /users/{owner}` → `Organization` ⇒ comment source `org`, else `account`.
- List repos: org ⇒ `/orgs/{o}/repos?type=all`; else if authenticated login == owner ⇒ `/user/repos?affiliation=owner`; else if App token ⇒ `/installation/repositories` filtered by owner; else `/users/{o}/repos?type=owner`. All paginated via `Link: rel="next"`.
- For each repo, compute overrides vs defaults (description/homepage included when non-empty; `archived` only when true; `visibility` derived from `private`).
- Stale detection: `repos/*.yml` stems with no live repo are candidates; verify each via `GET /repos/{o}/{name}` — 404/301 ⇒ gone, 200 ⇒ **abort with exit 1 before any write** (token-access guard), other status ⇒ raise.
- Then write `repos/<name>.yml` for every live repo and delete confirmed-gone files.

### `create-pr --settings-path <.github> --branch <safe-settings/snapshot> --base <main>`
- `git config` bot identity (`github-actions[bot]` / `41898282+github-actions[bot]@users.noreply.github.com`), `git checkout -B <branch>`, `git add --all <settings-path>/repos`.
- If nothing staged (`git diff --cached --quiet`) ⇒ print "No changes to commit." and return.
- Else commit `chore: snapshot live repo settings`, `git push --force origin <branch>`; if no open PR for head (`gh pr list`) ⇒ `gh pr create` with fixed title/body.

### `cleanup-runs [--owner <o>] [--repo <r>] --days <30> --keep <5> [--dry-run]`
- Resolve target: explicit `--owner` (with optional `--repo`); else parse `GITHUB_REPOSITORY=owner/name` (error if unset).
- Cutoff = now − days (UTC, `YYYY-MM-DD`). Repos = `[repo]` or all under owner.
- Per repo: list `status=completed&created=<cutoff` runs; group by `workflow_id`, sort newest-first, keep `keep`, delete the rest (or print `[dry-run]`). Print per-repo and summary lines (exact format preserved).

### Output format (`repos/<name>.yml`)
Header comment `# {owner}/{name} — overrides from {source} defaults`, blank line, optional `_fork: true` + blank, then `repository: {}` or `repository:` with `  key: value` lines; `description`/`homepage` double-quoted, bools as `true`/`false`.

## Requirements (testable)

- **R1** Module `github.com/nicerobot/tools.admin`; binary `tools.admin` (entrypoint unchanged); urfave/cli/v3 root with the 3 subcommands + identical flags/defaults.
- **R2** Layout: `cmd/tools.admin/`, `internal/{adminerr,githubapi,gitsvc,settings,overrides,snapshot,createpr,cleanupruns}`; generated nothing.
- **R3** DI: HTTP client behind an injected `Doer` interface; git/gh behind an injected command `Runner`; clock injected for cutoff. Every branch reachable from tests ⇒ **100% statement coverage**.
- **R4** Errors: one `adminerr.Error string` sentinel set; no `errors.New`/`fmt.Errorf` except `%w` wraps of sentinels. Failure paths asserted with `errors.Is`.
- **R5** Named param types (no bare `string`/`int` domain params); value receivers; private-by-default; cognitive complexity ≤ 7 (`gocognit -over 7` empty).
- **R6** `go.mod` `tool` stanza (gofumpt, staticcheck, govulncheck, gocognit, goreleaser) + `Makefile` `check` target (gofumpt -l, vet, staticcheck, govulncheck, gocognit, `go test` 100% cover-gate, goreleaser check) exits zero.
- **R7** Delete `src/`, `tests/`, `pyproject.toml`, `ruff.toml`, `mypy.ini`, `pytest.ini`, `uv.lock`, `.venv`, caches. Update `Dockerfile` (Go build) and `.github/workflows/ci.yml` (Go gate, floats Go minor with `check-latest`).

## Acceptance Criteria

- **AC1** `make check` exits zero: gofumpt clean, vet/staticcheck/govulncheck clean, gocognit ≤7 empty, **coverage 100.0%**, goreleaser check ok.
- **AC2** No `*.toml` remains in the repo (`find . -name '*.toml' -not -path './.git/*'` empty).
- **AC3** All three commands reproduce the documented output byte-for-byte (golden tests for `repos/*.yml`; table tests for run-pruning and target resolution).
- **AC4** `entrypoint.sh`/`action.yml`/`Dockerfile` invoke the `tools.admin` binary; the action still runs `snapshot`/`create-pr`/`cleanup-runs`.
