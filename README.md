# page

A small Go daemon that watches **qBittorrent** for completed downloads tagged
`book`, enriches them with metadata from **Hardcover**, and imports the EPUB/M4B
files into a **Calibre** library via `calibredb`.

## How it works

On each poll the daemon:

1. Logs into the qBittorrent WebUI and lists completed torrents carrying the
   configured tag (default `book`).
2. For each new torrent (tracked in a small state file so nothing is imported
   twice), finds the `.epub` / `.m4b` files.
3. Remaps the container paths reported by qBittorrent (it runs in Docker) to
   paths reachable from this host.
4. Extracts the ISBN, title and author:
   - **EPUB** — reads the OPF metadata (`dc:identifier`, `dc:title`, `dc:creator`).
   - **M4B/M4A** — walks the MP4 atom tree (`moov > udta > meta > ilst`) for the
     title/author tags and scans comment/description/freeform atoms for an ISBN.
5. Looks the book up on Hardcover — by ISBN first, falling back to a
   title/author search when no ISBN is found.
6. Imports the file with `calibredb add`, applying the enriched title, authors,
   ISBN, series, Hardcover identifier and cover.

> **Note on storage:** `calibredb add` copies the file into the Calibre library;
> Calibre has no hard-link import mode. The original stays in the download
> directory (so it keeps seeding), so the book occupies space in both places.

## Configuration

Copy `.env.example` to `.env` and fill it in. Real environment variables
override the `.env` file.

| Variable | Description |
| --- | --- |
| `QBIT_URL` | qBittorrent WebUI base URL |
| `QBIT_USERNAME` / `QBIT_PASSWORD` | WebUI credentials |
| `QBIT_TAG` | Tag to watch (default `book`); also applied to imported books |
| `PATH_MAP` | `container=host` prefix pairs, comma-separated. Longest prefix wins |
| `CALIBRE_LIBRARY` | Path to the Calibre library folder (with `metadata.db`) |
| `CALIBREDB_BIN` | `calibredb` binary (default `calibredb`) |
| `ADD_DUPLICATES` | Add even if Calibre flags a duplicate (default `false`) |
| `HARDCOVER_TOKEN` | Hardcover API token from <https://hardcover.app/account/api> |
| `HARDCOVER_URL` | GraphQL endpoint (default `https://api.hardcover.app/v1/graphql`) |
| `POLL_INTERVAL` | How often to poll (Go duration, default `5m`) |
| `STATE_FILE` | Where processed torrent hashes are recorded (default `state.json`) |
| `DRY_RUN` | Log the `calibredb` command instead of running it |

### Docker path remapping

qBittorrent in Docker reports paths inside its container. If the container sees
downloads at `/downloads` and that same volume is at `/mnt/user/downloads` on the
host running this daemon:

```
PATH_MAP=/downloads=/mnt/user/downloads
```

## Usage

```sh
go build -o page .

./page            # run as a daemon, polling every POLL_INTERVAL
./page -once      # process everything once and exit (good for cron)
./page -env /etc/page/.env
```

Set `DRY_RUN=true` first to see exactly which `calibredb add` commands would run
without touching the library.

## Docker

The image bundles `calibredb` (via Calibre's official headless installer) so you
don't need Calibre on the host.

```sh
docker build -t page .
```

The container exposes three mount points:

| Path | Purpose |
| --- | --- |
| `/config` | Holds `.env` and `state.json` (working dir) |
| `/calibre` | Your Calibre library (`CALIBRE_LIBRARY` defaults here) |
| `/downloads` | Where the daemon reads completed files |

`PATH_MAP` must remap the paths qBittorrent reports to paths **inside this
container**. If qBittorrent sees downloads at `/downloads` and you mount the same
data at `/downloads` here, then `PATH_MAP=/downloads=/downloads`.

```sh
docker run -d --name page \
  -v /mnt/user/appdata/page:/config \
  -v /mnt/user/calibre:/calibre \
  -v /mnt/user/downloads:/downloads \
  page
```

Or with compose:

```yaml
services:
  page:
    build: .
    container_name: page
    restart: unless-stopped
    volumes:
      - ./config:/config        # put your .env here
      - /mnt/user/calibre:/calibre
      - /mnt/user/downloads:/downloads
    environment:
      QBIT_URL: http://qbittorrent:8080
      QBIT_USERNAME: admin
      QBIT_PASSWORD: changeme
      HARDCOVER_TOKEN: your-token
      PATH_MAP: /downloads=/downloads
```

Environment variables can be set inline (as above) or via `/config/.env`.

## Development

```sh
go test ./...
go vet ./...
```
