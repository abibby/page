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
FROM golang:1.26-bookworm AS build

WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
# Pull in the compiled UI so //go:embed dist/* has files to embed.
COPY --from=ui /ui/dist ./ui/dist
# CGO disabled -> fully static binary that runs on the slim runtime image.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/page .

# --- Stage 3: runtime with calibredb ----------------------------------------
FROM debian:bookworm-slim

# Runtime libraries Calibre's bundled Qt needs to load, even for the CLI tools.
# Calibre is installed via its official isolated installer (current release,
# self-contained in /opt/calibre).
RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates \
        wget \
        xz-utils \
        python3 \
        libfontconfig1 \
        libgl1 \
        libegl1 \
        libopengl0 \
        libglx0 \
        libglib2.0-0 \
        libxkbcommon0 \
        libxcb-cursor0 \
        libxcb-xinerama0 \
        libxcb-icccm4 \
        libxcb-image0 \
        libxcb-keysyms1 \
        libxcb-randr0 \
        libxcb-render-util0 \
        libxcb-shape0 \
        libxcomposite1 \
        libxdamage1 \
        libxrandr2 \
        libxi6 \
        libxtst6 \
        libnss3 \
    && wget -nv -O- https://download.calibre-ebook.com/linux-installer.sh \
        | sh /dev/stdin isolated=y \
    && rm -rf /var/lib/apt/lists/*

# calibredb and friends live in /opt/calibre; run Qt headless.
ENV PATH="/opt/calibre:${PATH}" \
    QT_QPA_PLATFORM=offscreen \
    HOME=/config \
    STATE_FILE=/config/state.json \
    CALIBRE_LIBRARY=/calibre

# /config holds .env + state.json; mount your library and downloads.
WORKDIR /config
RUN mkdir -p /config /calibre /downloads
VOLUME ["/config", "/calibre", "/downloads"]

COPY --from=build /out/page /usr/local/bin/page

ENTRYPOINT ["/usr/local/bin/page"]