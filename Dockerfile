# Container image for the tools.admin GitHub Action. The CLI is a Go binary
# (radm); the runtime additionally needs git (snapshot/create-pr operate on a
# checkout) and gh (create-pr opens the PR). entrypoint.sh invokes
# `radm <subcommand>` from PATH.
FROM golang:1.26 AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/radm ./cmd/radm

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends git ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# install gh CLI
ADD https://github.com/cli/cli/releases/download/v2.65.0/gh_2.65.0_linux_amd64.deb /tmp/gh.deb
RUN dpkg -i /tmp/gh.deb && rm /tmp/gh.deb

COPY --from=builder /out/radm /usr/local/bin/radm
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
