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

### `create-pr --settings-path <.github> --branch <settings-sync/snapshot> --base <main>`

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

The layout MUST mirror the canonical [`gomatic/template.cli`](../../../../gomatic/template.cli) three-tier contract (**app → domain → implementation**), file-for-file — not an ad-hoc structure.

- **R1** Module `github.com/nicerobot/tools.admin`; **binary `radm`** (`cmd/radm/`); `entrypoint.sh`/`Dockerfile` invoke `radm`; urfave/cli/v3 root with the 3 subcommands + identical flags/defaults.
- **R2** Three-tier layout mirroring `template.cli`: app tier `internal/app/{action,output,logger,handler}.go` + `internal/app/commands/{snapshot,createpr,cleanupruns}/command.go`; domain tier `internal/domain/{snapshot,createpr,cleanupruns}/{package,config,run}.go`; implementation tier concept-named `internal/{repo,github,gitcmd,settings,overrides}`; sentinels in `internal/constants/{type,errors}.go`.
- **R3** DI via the template's package-var seam (`var deps = …` swapped in tests): HTTP behind an injected `Doer`; git/gh behind an injected runner; clock injected for cutoff. Every branch reachable ⇒ **100% statement coverage** (incl. `cmd/radm/main.go`).
- **R4** Errors: one `constants.Error string` sentinel set (copied verbatim from `template.cli`); no `errors.New`/`fmt.Errorf` except `%w` wraps of sentinels. Failure paths asserted with `errors.Is`.
- **R5** Named param types in `internal/repo` (no bare `string`/`int` domain params); value receivers; private-by-default; cognitive complexity ≤ 7.
- **R6** Output: each command's domain `Run` returns a structured `Result`, rendered as JSON via `app.Default` (uniform with `template.cli`'s `greet` — no bespoke progress strings). The `repos/<name>.yml` FILE format is still byte-for-byte preserved.
- **R7** Canonical tooling copied verbatim: the shared go-make `Makefile` (with a `Makefile.local` clearing the package-less `go-tooling/` submodule from the fan-out), `tools.repository`'s `.golangci.yaml` (the git/gh-CLI config: gosec `G115`+`G204` excluded), full `go.mod` `tool` stanza, `.goreleaser.yml` (`main: ./cmd/radm`). `make check` exits zero.
- **R8** Delete all Python (`src/`, `tests/`, `pyproject.toml`, `ruff.toml`, `mypy.ini`, `pytest.ini`, `uv.lock`, `.venv`, caches). Update `Dockerfile` (Go build of `radm`) and `ci.yml` (Go gate, Go minor floated with `check-latest`).

## Acceptance Criteria

- **AC1** `make check SUBMODULES=` exits zero: golangci-lint **0 issues**, staticcheck/govulncheck clean, **coverage 100.0%**; `goreleaser check` ok.
- **AC2** No `*.toml` remains in the repo (`find . -name '*.toml' -not -path './.git/*'` empty).
- **AC3** `make test-integration` (the `//go:build integration` tests) passes; golden tests reproduce `repos/*.yml` byte-for-byte; table tests cover run-pruning and target resolution. A live smoke-run (`radm cleanup-runs --dry-run` against a fake API) emits the JSON `Result`.
- **AC4** `internal/` tier shape matches `template.cli`; `entrypoint.sh`/`Dockerfile` invoke `radm`; the action still maps `snapshot`/`create-pr`/`cleanup-runs`.
