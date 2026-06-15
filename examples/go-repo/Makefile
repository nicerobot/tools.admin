# Local-development Makefile.
#
# You don't install any Go tooling. Every target is run inside the pinned
# go-tooling image against this checkout — so `make check`, `make lint`,
# `make build`, ... use the exact same tools and versions as CI, which runs
# `make -f ci.mk ...` inside the same image.
#
# Pin to a tag (v2) or, for byte-for-byte reproducibility, a digest:
#   GO_TOOLING_IMAGE = ghcr.io/nicerobot/admin-tools/go-tooling@sha256:<digest>
GO_TOOLING_IMAGE ?= ghcr.io/nicerobot/admin-tools/go-tooling:v2
DOCKER           ?= docker

.PHONY: pull
pull: ## Pull/update the pinned toolchain image
	$(DOCKER) pull $(GO_TOOLING_IMAGE)

.PHONY: shell
shell: ## Open a shell in the toolchain image
	$(DOCKER) run --rm -it -v "$(CURDIR):/src" -w /src $(GO_TOOLING_IMAGE) bash

# Catch-all: run any ci.mk target (check, lint, test, build, ...) in the image.
%:
	$(DOCKER) run --rm -v "$(CURDIR):/src" -w /src $(GO_TOOLING_IMAGE) make -f ci.mk $@

.DEFAULT_GOAL := check
