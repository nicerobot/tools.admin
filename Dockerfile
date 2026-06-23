FROM python:3.12-slim AS builder

WORKDIR /build
COPY pyproject.toml .
COPY src/ src/
RUN pip install --no-cache-dir --prefix=/install .

FROM python:3.12-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends git \
    && rm -rf /var/lib/apt/lists/*

# install gh CLI
ADD https://github.com/cli/cli/releases/download/v2.65.0/gh_2.65.0_linux_amd64.deb /tmp/gh.deb
RUN dpkg -i /tmp/gh.deb && rm /tmp/gh.deb

COPY --from=builder /install /usr/local
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
