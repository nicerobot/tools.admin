# Drop-in Makefile for a Go repository that uses the shared go-tooling image.
#
# The same image tag is the single source of truth for tool versions, used
# identically in CI and on developer laptops:
#
#   * In CI (job runs `container: .../go-tooling:v2`) or inside `docker run`,
#     /opt/go-tooling/tools.mk exists, so we include it directly and call its
#     targets (lint, test, check, ...).
#   * Locally (no toolchain installed), every target is transparently re-run
#     inside the pinned image, so `make lint` uses the exact same tool versions
#     as CI without installing anything.
#
# Pin to a tag (v2) or, for byte-for-byte reproducibility, a digest:
#   GO_TOOLING_IMAGE = ghcr.io/nicerobot/admin-tools/go-tooling@sha256:<digest>
GO_TOOLING_IMAGE ?= ghcr.io/nicerobot/admin-tools/go-tooling:v2

ifneq ($(wildcard /opt/go-tooling/tools.mk),)
# ---- Running inside the go-tooling image -------------------------------
include /opt/go-tooling/tools.mk

# Your own targets can reuse the bundled ones as prerequisites.
.PHONY: build
build: lint test ## Build after linting and testing
	go build ./...

.DEFAULT_GOAL := check

else
# ---- Running on a developer machine (no toolchain installed) -----------
# Proxy any target into the pinned image so local == CI tool versions.
DOCKER ?= docker

.PHONY: pull
pull: ## Pull/update the pinned toolchain image
	$(DOCKER) pull $(GO_TOOLING_IMAGE)

.PHONY: shell
shell: ## Open a shell in the toolchain image
	$(DOCKER) run --rm -it -v "$(CURDIR):/src" -w /src $(GO_TOOLING_IMAGE) bash

# Catch-all: `make lint`, `make check`, `make build`, ... run inside the image.
%:
	$(DOCKER) run --rm -v "$(CURDIR):/src" -w /src $(GO_TOOLING_IMAGE) make $@

.DEFAULT_GOAL := check
endif
