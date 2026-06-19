# syntax=docker/dockerfile:1

# Multi-stage build for easy-infra.
#
# Stage 1 builds the Vite/React frontend into ui/dist. Stage 2 compiles the Go
# binary, which embeds that bundle via //go:embed (see ui/embed.go). The final
# stage ships only the static binary on a minimal base. SQLite is pure-Go
# (modernc.org/sqlite), so the build needs no cgo and the binary is fully static.

# ---- Stage 1: build the frontend bundle ----
FROM node:22-alpine AS ui
WORKDIR /app/ui

# Install dependencies against the lockfile first so this layer caches across
# source-only changes.
COPY ui/package.json ui/package-lock.json ./
RUN npm ci

# Build the production bundle into ui/dist (consumed by the Go embed below).
COPY ui/ ./
RUN npm run build

# ---- Stage 2: build the Go binary ----
FROM golang:1.25-alpine AS build
WORKDIR /src

# Download modules first for layer caching.
COPY go.mod go.sum ./
RUN go mod download

# Bring in the rest of the source, then drop in the freshly built UI bundle so
# `//go:embed all:dist` picks it up.
COPY . .
COPY --from=ui /app/ui/dist ./ui/dist

# CGO_ENABLED=0 yields a static binary (modernc.org/sqlite is pure Go).
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/easy-infra .

# ---- Stage 3: minimal runtime image ----
FROM alpine:3.20

# ca-certificates lets the app reach TLS endpoints (minio, AWS SDK, etc.).
RUN apk add --no-cache ca-certificates \
    && adduser -D -u 10001 easyinfra

COPY --from=build /out/easy-infra /usr/local/bin/easy-infra

# Persist the tool-owned SQLite store outside the container.
ENV EASY_INFRA_CONFIG_DIR=/data
RUN mkdir -p /data && chown easyinfra:easyinfra /data
VOLUME /data

USER easyinfra
EXPOSE 8080

# Serve on the default port 8080. Override the port at runtime with
# `docker run … easy-infra --port 9000`.
ENTRYPOINT ["easy-infra"]
CMD ["serve"]
