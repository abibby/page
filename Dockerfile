# syntax=docker/dockerfile:1

# --- Stage 1: build the UI bundle -------------------------------------------
FROM node:22-bookworm-slim AS ui
WORKDIR /ui
# Install deps first so this layer caches unless the lockfile changes.
COPY ui/package.json ui/package-lock.json ./
RUN npm ci
COPY ui/ ./
# Produces /ui/dist, which the Go binary embeds via //go:embed dist/*.
RUN npm run build

# --- Stage 2: build a static Go binary --------------------------------------
FROM golang:1.27-bookworm AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Pull in the compiled UI so //go:embed dist/* has files to embed.
COPY --from=ui /ui/dist ./ui/dist
# CGO disabled -> fully static binary that runs on the slim runtime image.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/page .

# --- Stage 3: runtime ------------------------------------------------------
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Keep the existing mount path and environment variable compatible.
ENV HOME=/config \
    STATE_FILE=/config/state.json \
    DATABASE_PATH=/config/db.sqlite \
    CALIBRE_LIBRARY=/calibre

WORKDIR /config
RUN mkdir -p /config /calibre /downloads
VOLUME ["/config", "/calibre", "/downloads"]

COPY --from=build /out/page /usr/local/bin/page
ENTRYPOINT ["/usr/local/bin/page"]
