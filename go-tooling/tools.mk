# tools.mk — Go quality toolchain targets.
#
# This file is shipped inside the go-tooling image at /opt/go-tooling/tools.mk.
# Every tool it invokes is precompiled into the image and on $PATH, so a
# consuming repository needs none of them in its own go.mod.
#
# Use it one of two ways from a repo running inside the image:
#
#   1. Directly:   make -f /opt/go-tooling/tools.mk check
#   2. Included:   add `include /opt/go-tooling/tools.mk` to your Makefile
#
# All knobs below are overridable, e.g. `make -f tools.mk lint GO_PKGS=./cmd/...`.

# Directory this makefile lives in — used to locate shipped default configs.
# Resolves correctly whether run with -f or via `include`.
GO_TOOLING_DIR ?= $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))

GO              ?= go
GO_PKGS         ?= ./...
GO_TEST_FLAGS   ?= -race -count=1
COVER_PROFILE   ?= coverage.out

# Complexity / duplication thresholds.
GOCYCLO_OVER    ?= 15
GOCOGNIT_OVER   ?= 20
DUPL_THRESHOLD  ?= 100

# Config files: prefer a repo-local copy, fall back to the image defaults.
GOLANGCI_CONFIG ?= $(firstword $(wildcard .golangci.yml .golangci.yaml .golangci.toml) $(GO_TOOLING_DIR)/.golangci.yml)
REVIVE_CONFIG   ?= $(firstword $(wildcard revive.toml .revive.toml) $(GO_TOOLING_DIR)/revive.toml)

.DEFAULT_GOAL := help

# ---------------------------------------------------------------------------
# Meta
# ---------------------------------------------------------------------------

.PHONY: help
help: ## List available targets
	@awk 'BEGIN{FS=":.*##"} /^[a-zA-Z0-9_-]+:.*##/{printf "  \033[36m%-16s\033[0m %s\n",$$1,$$2}' $(MAKEFILE_LIST) | sort

.PHONY: tool-versions
tool-versions: ## Print the version of every bundled tool
	@echo "go:           $$($(GO) version | awk '{print $$3}')"
	@echo "golangci-lint:$$(golangci-lint version --short 2>/dev/null || golangci-lint --version)"
	@echo "staticcheck:  $$(staticcheck --version)"
	@echo "govulncheck:  $$(govulncheck -version 2>/dev/null | tail -1)"
	@echo "gosec:        $$(gosec --version 2>/dev/null | awk '/Version/{print $$2}')"
	@echo "gofumpt:      $$(gofumpt --version)"
	@echo "revive:       $$(revive --version 2>/dev/null | head -1)"

# ---------------------------------------------------------------------------
# Formatting
# ---------------------------------------------------------------------------

.PHONY: fmt
fmt: ## Format code (gofumpt, goimports, gci)
	gofumpt -extra -w .
	goimports -w .
	gci write --skip-generated -s standard -s default -s localmodule .

.PHONY: fmt-check
fmt-check: ## Fail if any file is not formatted
	@out="$$(gofumpt -extra -l .)"; \
	if [ -n "$$out" ]; then echo "not gofumpt-formatted:"; echo "$$out"; exit 1; fi
	@out="$$(goimports -l .)"; \
	if [ -n "$$out" ]; then echo "imports not formatted:"; echo "$$out"; exit 1; fi

# ---------------------------------------------------------------------------
# Modules
# ---------------------------------------------------------------------------

.PHONY: tidy
tidy: ## Run go mod tidy
	$(GO) mod tidy

.PHONY: tidy-check
tidy-check: ## Fail if go.mod/go.sum are not tidy
	@cp go.mod go.mod.bak; cp go.sum go.sum.bak 2>/dev/null || true; \
	$(GO) mod tidy; \
	if ! diff -q go.mod go.mod.bak >/dev/null 2>&1 || { [ -f go.sum.bak ] && ! diff -q go.sum go.sum.bak >/dev/null 2>&1; }; then \
	  echo "go.mod/go.sum are not tidy — run 'make tidy'"; rc=1; else rc=0; fi; \
	mv go.mod.bak go.mod; [ -f go.sum.bak ] && mv go.sum.bak go.sum; exit $$rc

# ---------------------------------------------------------------------------
# Linting & static analysis
# ---------------------------------------------------------------------------

.PHONY: vet
vet: ## go vet
	$(GO) vet $(GO_PKGS)

.PHONY: lint
lint: ## golangci-lint (aggregate linters)
	golangci-lint run --config $(GOLANGCI_CONFIG) $(GO_PKGS)

.PHONY: lint-fix
lint-fix: ## golangci-lint with --fix
	golangci-lint run --config $(GOLANGCI_CONFIG) --fix $(GO_PKGS)

.PHONY: staticcheck
staticcheck: ## staticcheck
	staticcheck $(GO_PKGS)

.PHONY: revive
revive: ## revive style linter
	revive -config $(REVIVE_CONFIG) -set_exit_status $(GO_PKGS)

.PHONY: errcheck
errcheck: ## errcheck (unchecked errors)
	errcheck $(GO_PKGS)

.PHONY: ineffassign
ineffassign: ## ineffassign (ineffectual assignments)
	ineffassign $(GO_PKGS)

.PHONY: misspell
misspell: ## misspell (common misspellings)
	misspell -error .

.PHONY: deadcode
deadcode: ## deadcode (unreachable functions)
	deadcode $(GO_PKGS)

.PHONY: cyclo
cyclo: ## cyclomatic complexity (gocyclo)
	gocyclo -over $(GOCYCLO_OVER) .

.PHONY: cognit
cognit: ## cognitive complexity (gocognit)
	gocognit -over $(GOCOGNIT_OVER) .

.PHONY: complexity
complexity: cyclo cognit ## cyclomatic + cognitive complexity

.PHONY: dupl
dupl: ## duplicate code detection
	dupl -threshold $(DUPL_THRESHOLD) .

.PHONY: nilaway
nilaway: ## nil-panic analysis (Uber NilAway)
	nilaway $(GO_PKGS)

# ---------------------------------------------------------------------------
# Security & vulnerabilities
# ---------------------------------------------------------------------------

.PHONY: vulncheck
vulncheck: ## govulncheck (known vulnerabilities)
	govulncheck $(GO_PKGS)

.PHONY: gosec
gosec: ## gosec (security analyzer)
	gosec -quiet $(GO_PKGS)

.PHONY: security
security: gosec vulncheck ## gosec + govulncheck

# ---------------------------------------------------------------------------
# Tests & coverage
# ---------------------------------------------------------------------------

.PHONY: test
test: ## Run tests (gotestsum)
	gotestsum --format pkgname -- $(GO_TEST_FLAGS) $(GO_PKGS)

.PHONY: cover
cover: ## Run tests with coverage profile
	gotestsum --format pkgname -- $(GO_TEST_FLAGS) -coverprofile=$(COVER_PROFILE) -covermode=atomic $(GO_PKGS)
	$(GO) tool cover -func=$(COVER_PROFILE) | tail -1

.PHONY: cover-html
cover-html: cover ## Generate HTML coverage report (coverage.html)
	$(GO) tool cover -html=$(COVER_PROFILE) -o coverage.html

# ---------------------------------------------------------------------------
# Aggregate targets
# ---------------------------------------------------------------------------

.PHONY: analyze
analyze: vet staticcheck complexity deadcode ## Static analysis bundle

.PHONY: check
check: fmt-check lint analyze security test ## Full quality gate (the CI target)

.PHONY: ci
ci: check ## Alias for `check`
