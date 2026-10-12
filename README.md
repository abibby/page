# Page

Page stores its book catalogue in SQLite using Salusa's database connection,
transactions, and migrations. Calibre and `calibredb` are no longer required.

Set `DATABASE_PATH` to Page's SQLite database (default `./db.sqlite`) and
`LIBRARY_PATH` to the directory containing the library files. Existing
`CALIBRE_LIBRARY` settings remain supported when `LIBRARY_PATH` is unset.
The Docker image keeps the existing `/config`, `/calibre`, and `/downloads`
mounts, with the catalogue at `/config/db.sqlite`.

When Page first opens a library containing a Calibre `metadata.db`, it imports
that catalogue into its own database in one transaction. Book IDs, identifiers,
metadata, and existing file paths are preserved. The source database is opened
read-only, and existing folders and files are neither renamed nor moved. Audio
tracks stored alongside books by older Page versions are included too. A failed
import rolls back and is retried when the library is next opened.

Keep `DATABASE_PATH` separate from the original `metadata.db`. After migration,
Page's database is authoritative: later changes made through Calibre are not
synchronized. Persist and back up both Page's database and the library files.
An existing Page catalogue cannot be combined automatically with another Calibre
catalogue because their book IDs may overlap.

New authors use their names as folder names, for example `Ada Author/Book (43)`.
Existing author paths such as `Ada Author (9)` are reused, including when adding
another book by that author. Characters that cannot safely form a filename are
replaced with underscores. Book directories retain an ID suffix to distinguish
books with the same title. Existing ebook formats are preserved on repeat
imports; audiobook tracks are stored by filename.

Every five minutes, `page:scan` runs a library scan to register book folders that
exist under `LIBRARY_PATH` but are missing from Page's catalogue. The scan expects
`Author/Book/` folders, includes audio tracks in subfolders, and leaves all files
and paths in place. It reads `metadata.opf` or embedded EPUB metadata, falling back
to folder names when metadata is unavailable. It does not need Hardcover or
qBittorrent. Indexed book folders are skipped, so scans do not overwrite catalogue
edits or synchronize new formats into existing books. Symlinks and unrelated files
are ignored; failed folders are reported and retried on the next scan. `DRY_RUN`
prevents the scan from writing catalogue entries.

Book search supports plain text and the `title:`, `authors:`, `series:`, `isbn:`,
`tags:`, `id:`, and `identifiers:type:value` prefixes. Calibre's advanced Boolean
search expressions and custom columns are not imported or interpreted.

Run the checks with:

```sh
go test ./...
golangci-lint run ./...
CGO_ENABLED=0 go build .
```
