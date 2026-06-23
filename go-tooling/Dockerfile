# syntax=docker/dockerfile:1
#
# Go quality toolchain image.
#
# Stage 1 compiles every tool declared in go.mod's `tool` block into static
# binaries. Stage 2 is the full official Go image (consumers still need `go`
# itself to build/test/vet) with those binaries on $PATH plus the includable
# tools.mk and default configs under /opt/go-tooling.

# Latest official Go image. The bundled tools require Go >= 1.25.
# Override at build time: docker build --build-arg GO_VERSION=1.26 ...
ARG GO_VERSION=1.26

FROM golang:${GO_VERSION} AS tools

ENV GOBIN=/tools/bin \
    CGO_ENABLED=0
WORKDIR /src

# Resolve the module graph first so it caches independently of source changes.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Compile every tool in the `tool` block into $GOBIN in one shot.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go install tool

# ---------------------------------------------------------------------------
FROM golang:${GO_VERSION}

LABEL org.opencontainers.image.source="https://github.com/nicerobot/tools.admin"
LABEL org.opencontainers.image.description="Go quality toolchain: formatters, linters, static & complexity analysis, security and vulnerability scanners precompiled with an includable Makefile."

# make (to drive tools.mk) + a C toolchain (race detector / cgo) + certs.
RUN apt-get update \
    && apt-get install -y --no-install-recommends make gcc ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Precompiled tool binaries — /go/bin is already on PATH in the golang image.
COPY --from=tools /tools/bin/ /go/bin/

# Includable makefile, default linter configs, and the action entrypoint.
ENV GO_TOOLING_DIR=/opt/go-tooling
COPY tools.mk .golangci.yml entrypoint.sh ${GO_TOOLING_DIR}/
RUN chmod +x ${GO_TOOLING_DIR}/entrypoint.sh

# Consuming CI checks out the repo here (Docker GitHub Actions override this
# with /github/workspace automatically).
WORKDIR /src
