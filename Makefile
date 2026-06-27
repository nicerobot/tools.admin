# Self-contained quality gate for tools.admin. Every tool is pinned in the
# go.mod `tool (...)` stanza and run via `go tool <name>`, so there are no global
# installs and CI needs only the Go toolchain. `make check` is the full gate and
# must exit zero before any change is considered complete.
#
# Coverage is measured over OWNED packages only — cmd/ is the composition root
# (real HTTP/subprocess/filesystem wiring) and is excluded, matching the policy
# in the reference repo (nicerobot/tools.repository).

GO ?= go

GOFUMPT     := $(GO) tool gofumpt
STATICCHECK := $(GO) tool staticcheck
GOVULNCHECK := $(GO) tool govulncheck
GOCOGNIT    := $(GO) tool gocognit
GORELEASER  := $(GO) tool goreleaser

COVERAGE_FOLDER := var
COVER_THRESHOLD := 100.0%
COVER_PKGS      := $(shell $(GO) list ./... | grep -v /cmd/)
comma           := ,
empty           :=
space           := $(empty) $(empty)
COVERPKG        := $(subst $(space),$(comma),$(strip $(COVER_PKGS)))

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

.PHONY: check
check: fmt-check vet staticcheck vulncheck gocognit cover goreleaser-check ## Run the full quality gate

.PHONY: fmt
fmt: ## Format all Go files
	$(GOFUMPT) -w .

.PHONY: fmt-check
fmt-check: ## Fail if any file is not gofumpt-clean
	@out=$$($(GOFUMPT) -l .); test -z "$$out" || { echo "gofumpt needs to run on:"; echo "$$out"; exit 1; }

.PHONY: vet
vet: ## Run go vet
	$(GO) vet ./...

.PHONY: staticcheck
staticcheck: ## Run staticcheck
	$(STATICCHECK) ./...

.PHONY: vulncheck
vulncheck: ## Run govulncheck
	$(GOVULNCHECK) ./...

.PHONY: gocognit
gocognit: ## Fail if any production function exceeds cognitive complexity 7
	@out=$$($(GOCOGNIT) -over 7 $(shell $(GO) list -f '{{.Dir}}' ./...)); test -z "$$out" || { echo "cognitive complexity over 7:"; echo "$$out"; exit 1; }

$(COVERAGE_FOLDER):
	mkdir -p $@

.PHONY: cover
cover: $(COVERAGE_FOLDER) ## Run tests and assert 100% statement coverage of owned packages
	$(GO) test -covermode=atomic -coverpkg=$(COVERPKG) -coverprofile=$(COVERAGE_FOLDER)/coverage.out $(COVER_PKGS)
	@total=$$($(GO) tool cover -func=$(COVERAGE_FOLDER)/coverage.out | awk '/^total:/{print $$3}'); \
		echo "total coverage: $$total"; \
		test "$$total" = "$(COVER_THRESHOLD)" || { echo "coverage $$total below $(COVER_THRESHOLD):"; $(GO) tool cover -func=$(COVERAGE_FOLDER)/coverage.out | awk '$$3 != "100.0%"'; exit 1; }

.PHONY: goreleaser-check
goreleaser-check: ## Validate the goreleaser config
	$(GORELEASER) check

.PHONY: build
build: ## Build a snapshot binary for the current platform
	$(GORELEASER) build --single-target --snapshot --clean

.PHONY: clean
clean: ## Remove build and coverage artifacts
	rm -rf dist $(COVERAGE_FOLDER)

.PHONY: push
push: ## Tag v2 and force-push main + the v2 action tag
	git tag -f v2 \
	&& git push -f origin main \
	&& git push -f origin v2

.PHONY: save
save: ## Autosquash fixup commits from the root
	git cfx \
	&& GIT_SEQUENCE_EDITOR=: git rebase --root --autosquash
