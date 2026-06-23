# CI makefile — runs INSIDE the go-tooling image, which provides the toolchain
# and tools.mk. The workflow invokes it as `make -f ci.mk check`.
#
# This is also what local development proxies into the image (see Makefile),
# so CI and local runs share the exact same targets and tool versions.
include /opt/go-tooling/tools.mk

# Project targets can reuse the bundled ones as prerequisites.
.PHONY: build
build: lint test ## Build after linting and testing
	go build ./...

.DEFAULT_GOAL := check
