# syntax=docker/dockerfile:1

# ── Frontend: Vue SPA → /src/web/dist ────────────────────────────────────────
# Runs on the build platform: the output is plain JS/CSS, the same for every target.
FROM --platform=$BUILDPLATFORM node:24-alpine AS frontend
WORKDIR /src/frontend
# frontend/vendor holds the axios tarball that package.json points to.
COPY frontend/package.json frontend/package-lock.json ./
COPY frontend/vendor ./vendor
RUN --mount=type=cache,target=/root/.npm npm ci
COPY frontend/ ./
RUN npm run build

# ── Backend: static Go binary with the SPA embedded ──────────────────────────
# Also on the build platform; Go cross-compiles, so no emulation is needed.
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
COPY --from=frontend /src/web/dist ./web/dist

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath \
      -ldflags "-s -w -X go-cubemail/cmd.Version=$VERSION -X go-cubemail/cmd.BuildDate=$BUILD_DATE -X go-cubemail/cmd.GitCommit=$GIT_COMMIT" \
      -o /out/go-cubemail .

# Default config with the relative paths moved into the /data volume. The grep
# fails the build if the defaults change and the rewrite stops matching.
RUN mkdir /out/data \
 && sed -e 's|"\./data/app\.db"|"/data/app.db"|' -e 's|"\./tmp/uploads"|"/data/uploads"|' \
      web/files/config.default.toml > /out/config.toml \
 && grep -q '"/data/app.db"' /out/config.toml

# ── Runtime ──────────────────────────────────────────────────────────────────
# Alpine rather than distroless: the entrypoint needs a shell, and the base
# image already ships the CA bundle that IMAP/SMTP over TLS needs. This stage
# has no RUN step, so multi-arch builds need no QEMU.
FROM alpine:3.24

LABEL org.opencontainers.image.source="https://github.com/jniltinho/go-cubemail" \
      org.opencontainers.image.title="go-cubemail" \
      org.opencontainers.image.description="Self-hosted webmail (IMAP/SMTP, CalDAV, CardDAV, ActiveSync) in a single Go binary" \
      org.opencontainers.image.licenses="MIT"

COPY --from=build /out/go-cubemail /usr/local/bin/go-cubemail
COPY --from=build /out/config.toml /etc/go-cubemail/config.toml
COPY --from=build --chown=10001:10001 /out/data /data

COPY --chmod=755 <<'EOF' /usr/local/bin/docker-entrypoint.sh
#!/bin/sh
set -e
if [ "$*" = "serve" ]; then
  # The session password is AES-encrypted with this key; the app accepts the
  # published placeholder (and an empty key), so refuse to start with it.
  if [ -z "$GORC_SERVER_SECRET_KEY" ] && grep -q 'change-this-secret-key' /etc/go-cubemail/config.toml; then
    echo "error: set GORC_SERVER_SECRET_KEY (32 characters, e.g. \`openssl rand -base64 24\`)" \
         "or mount your own /etc/go-cubemail/config.toml" >&2
    exit 1
  fi
  # `serve` does not create or upgrade the schema; `migrate` is idempotent.
  go-cubemail migrate
fi
exec go-cubemail "$@"
EOF

USER 10001:10001
WORKDIR /data
VOLUME /data
EXPOSE 8080
ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["serve"]
