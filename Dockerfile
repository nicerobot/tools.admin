# Runtime image for the tools.admin GitHub Action. The radm binary is built once
# by goreleaser and COPYed in here — the image never compiles Go, so the published
# image and the released binary are the exact same artifact at the same version.
# The runtime additionally needs git (snapshot/create-pr operate on a checkout)
# and gh (create-pr opens the PR). goreleaser's `dockers:` config places the
# linux/amd64 radm binary in the build context. entrypoint.sh invokes
# `radm <subcommand>` from PATH.
FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends git ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# install gh CLI
ADD https://github.com/cli/cli/releases/download/v2.65.0/gh_2.65.0_linux_amd64.deb /tmp/gh.deb
RUN dpkg -i /tmp/gh.deb && rm /tmp/gh.deb

COPY radm-linux-amd64 /usr/local/bin/radm
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh /usr/local/bin/radm

ENTRYPOINT ["/entrypoint.sh"]
