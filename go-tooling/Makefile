# Maintenance makefile for the go-tooling image itself.
# (Consumers use tools.mk, not this file.)

IMAGE       ?= ghcr.io/nicerobot/admin-tools/go-tooling
TAG         ?= dev
GO_VERSION  ?= 1.26

.PHONY: help
help:
	@echo "build         Build the go-tooling image ($(IMAGE):$(TAG))"
	@echo "verify        Validate the shipped golangci-lint config"
	@echo "upgrade       Bump all bundled tools to latest and re-tidy"
	@echo "tool-versions Print bundled tool versions (requires built image)"
	@echo "demo          Run 'make check' against a throwaway sample module"

.PHONY: build
build:
	docker build --build-arg GO_VERSION=$(GO_VERSION) -t $(IMAGE):$(TAG) .

.PHONY: verify
verify: build
	docker run --rm $(IMAGE):$(TAG) golangci-lint config verify -c /opt/go-tooling/.golangci.yml

.PHONY: tool-versions
tool-versions:
	docker run --rm $(IMAGE):$(TAG) make -f /opt/go-tooling/tools.mk tool-versions

# Refresh every tool in the `tool` block to its latest release.
.PHONY: upgrade
upgrade:
	go get -tool $$(go list -m -f '{{if .Tool}}{{.Path}}{{end}}' all 2>/dev/null) 2>/dev/null || true
	go get tool@latest 2>/dev/null || true
	go mod tidy

.PHONY: demo
demo: build
	@tmp=$$(mktemp -d); \
	printf 'package main\n\nimport "fmt"\n\nfunc main() { fmt.Println("ok") }\n' > $$tmp/main.go; \
	( cd $$tmp && go mod init demo.example >/dev/null ); \
	docker run --rm -v $$tmp:/src -w /src $(IMAGE):$(TAG) \
	  make -f /opt/go-tooling/tools.mk fmt-check lint analyze; \
	rm -rf $$tmp
