# go-tooling

A prebuilt container image that bundles a full suite of Go quality tooling —
formatters, linters, static/complexity analysis, and security & vulnerability
scanners — together with an **includable Makefile** (`tools.mk`).

The point: **other repositories get the entire toolchain without embedding any
of it.** No `tool` directives, no `go install`, no pinned tool versions in your
`go.mod`. You run your CI inside this image (or call the bundled action) and use
its `make` targets.

```
ghcr.io/nicerobot/admin-tools/go-tooling:v2
```

## What's inside

Every tool is declared in a single [`go.mod`](go.mod) `tool` block and compiled
into the image with `go install tool` (Go 1.24+ tool dependencies). They live on
`$PATH` (`/go/bin`).

| Category | Tools |
|---|---|
| Format | `gofumpt`, `goimports`, `gci`, `golines` |
| Lint (aggregate) | `golangci-lint` (v2) |
| Style / correctness | `revive`, `errcheck`, `ineffassign`, `misspell`, `staticcheck` |
| Static analysis | `go vet`, `staticcheck`, `deadcode`, `nilaway` |
| Complexity | `gocyclo` (cyclomatic), `gocognit` (cognitive), `dupl` (duplication) |
| Security | `gosec` |
| Vulnerabilities | `govulncheck` |
| Tests / coverage | `gotestsum`, `go-junit-report`, `gocov`, `gocov-xml` |

`golangci-lint` additionally runs many of the above (and more) as integrated
linters; the standalone binaries are there for targeted, scriptable use.

## Make targets (`tools.mk`)

| Target | Does |
|---|---|
| `fmt` / `fmt-check` | apply / verify formatting |
| `tidy` / `tidy-check` | `go mod tidy` / verify tidy |
| `vet` | `go vet` |
| `lint` / `lint-fix` | `golangci-lint run` |
| `staticcheck`, `errcheck`, `ineffassign`, `misspell` | individual linters |
| `complexity` (`cyclo` + `cognit`), `dupl`, `deadcode`, `nilaway` | analysis |
| `vulncheck`, `gosec`, `security` | vulnerability & security scans |
| `test`, `cover`, `cover-html` | tests & coverage |
| `analyze` | `vet` + `staticcheck` + `complexity` + `deadcode` |
| `check` / `ci` | **full gate**: `fmt-check` + `lint` + `analyze` + `security` + `test` |

Run `make -f /opt/go-tooling/tools.mk help` in the image for the live list.

Every knob is overridable, e.g.:

```sh
make -f /opt/go-tooling/tools.mk lint GO_PKGS=./cmd/...
make -f /opt/go-tooling/tools.mk complexity GOCYCLO_OVER=20 GOCOGNIT_OVER=25
```

## Usage

### 1. As a GitHub Action (simplest)

```yaml
jobs:
  go-quality:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: nicerobot/admin-tools/go-tooling@v2
        with:
          target: check        # default
```

### 2. Run your job inside the image

```yaml
jobs:
  go-quality:
    runs-on: ubuntu-latest
    container: ghcr.io/nicerobot/admin-tools/go-tooling:v2
    steps:
      - uses: actions/checkout@v4
      - run: make -f /opt/go-tooling/tools.mk check
```

### 3. Include `tools.mk` from your own Makefile

When running inside the image, your repo's `Makefile` can pull the targets in
directly — no copy needed:

```makefile
include /opt/go-tooling/tools.mk

# add your own targets; reuse the bundled ones as prerequisites
build: lint test
	go build ./...
```

### 4. Locally via Docker

```sh
docker run --rm -v "$PWD:/src" -w /src \
  ghcr.io/nicerobot/admin-tools/go-tooling:v2 \
  make -f /opt/go-tooling/tools.mk check
```

A complete, copy-paste template (project `Makefile` + workflow) lives in
[`../examples/go-repo/`](../examples/go-repo/).

## Local development — same tool versions as CI

The **image tag is the single source of truth for tool versions.** Because the
toolchain is baked into the image, a developer machine uses the exact versions
CI uses simply by running the same image — nothing is installed on the host.

[`examples/go-repo/`](../examples/go-repo/) splits this into two makefiles:

- **[`ci.mk`](../examples/go-repo/ci.mk)** runs *inside* the image. It
  `include`s `/opt/go-tooling/tools.mk` and adds your project targets. CI calls
  it directly: `make -f ci.mk check`.
- **[`Makefile`](../examples/go-repo/Makefile)** is for laptops with no
  toolchain installed. Every target is transparently re-run inside the pinned
  image against `ci.mk`:

  ```makefile
  GO_TOOLING_IMAGE ?= ghcr.io/nicerobot/admin-tools/go-tooling:v2
  %:
  	docker run --rm -v "$(CURDIR):/src" -w /src $(GO_TOOLING_IMAGE) make -f ci.mk $@
  ```

So a developer just runs `make lint` / `make check` and gets identical results
to CI. `make shell` drops into the image; `make pull` updates it.

For byte-for-byte reproducibility, pin a **digest** instead of a tag and bump it
deliberately (e.g. via Dependabot/Renovate watching the image):

```makefile
GO_TOOLING_IMAGE = ghcr.io/nicerobot/admin-tools/go-tooling@sha256:<digest>
```

## Configuration

The image ships a single default config at `/opt/go-tooling/.golangci.yml` —
the golangci-lint **v2** config (curated linter set). `revive` runs as one of
golangci-lint's linters and is configured **in that YAML file** (the standalone
`revive` binary only reads TOML, which this avoids).

`tools.mk` prefers a **repo-local** config when present (`.golangci.yml`,
`.golangci.yaml`, or `.golangci.toml` in your repo root) and falls back to the
shipped default otherwise. So consumers can override without touching this image.

## Maintaining this image

```sh
make build           # build locally as :dev
make verify          # validate the shipped golangci-lint config
make demo            # run the gate against a throwaway module
make tool-versions   # print bundled tool versions
make upgrade         # bump all bundled tools and re-tidy
```

The image is published by [`release.yml`](../.github/workflows/release.yml) on
`v*` tags as `ghcr.io/nicerobot/admin-tools/go-tooling:<tag>` and `:latest`.
