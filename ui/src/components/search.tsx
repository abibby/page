import { useCallback, useState } from "react";
import {
  hardcoverSearch,
  torrentSearch,
  type HardcoverBook,
  type Torrent,
} from "../api/api";
import { useAsync } from "../hooks/use-async";
import { useActiveTorrents } from "../hooks/use-active-torrents";
import { Cover } from "./book-card";
import { BookSkeletons, EmptyState, ErrorNotice } from "./feedback";
import { Icon } from "./icon";
import { formatBytes } from "../lib/format";

export function BookSearch({
  search,
  onSelectBook,
  busy,
}: {
  search: string;
  onSelectBook: (book: HardcoverBook) => void;
  busy?: boolean;
}) {
  const [attempt, setAttempt] = useState(0);
  const books = useAsync(
    useCallback(
      (signal) => {
        void attempt;
        return search.trim()
          ? hardcoverSearch({ search, signal })
          : Promise.resolve([]);
      },
      [search, attempt],
    ),
  );
  if (!search.trim())
    return (
      <EmptyState title="What will you read next?" icon="search">
        <p>
          Search Hardcover by title or author.
          <br />
          Find a book you love and give it a home in your library.
        </p>
      </EmptyState>
    );
  if (books.loading) return <BookSkeletons />;
  if (books.error)
    return (
      <ErrorNotice
        error={books.error}
        retry={() => setAttempt((value) => value + 1)}
      />
    );
  if (!books.value.length)
    return (
      <EmptyState title="No matching books" icon="search">
        <p>Try a different title, author, or a shorter search.</p>
      </EmptyState>
    );
  return (
    <>
      <p className="results-label">
        {books.value.length} results from Hardcover
      </p>
      <ul className="book-grid discovery-grid">
        {books.value.map((book) => (
          <li key={book.id}>
            <button
              className="book-card discovery-card"
              disabled={busy}
              onClick={() => onSelectBook(book)}
            >
              <div className="book-cover-wrap">
                <Cover
                  title={book.title}
                  author={hardcoverAuthor(book)}
                  url={book.image?.url}
                />
                <span className="cover-open">
                  <Icon name="plus" size={18} />
                </span>
              </div>
              <div className="book-card-info">
                <h3>{book.title}</h3>
                <p>{hardcoverAuthor(book) || "Unknown author"}</p>
                <span className="add-book-label">
                  <Icon name="plus" size={14} />
                  Add to library
                </span>
              </div>
            </button>
          </li>
        ))}
      </ul>
    </>
  );
}

function hardcoverAuthor(book: HardcoverBook) {
  return (book.contributions ?? [])
    .map((contribution) => contribution.author?.name)
    .filter(Boolean)
    .join(" & ");
}

export function TorrentSearch({
  search,
  onSelectTorrent,
}: {
  search: string;
  onSelectTorrent: (torrent: Torrent) => Promise<void>;
}) {
  const [attempt, setAttempt] = useState(0);
  const [sortBySeeds, setSortBySeeds] = useState(false);
  const [pending, setPending] = useState<string>();
  const [queued, setQueued] = useState<string[]>([]);
  const [error, setError] = useState<unknown>();
  const active = useActiveTorrents();
  const torrents = useAsync(
    useCallback(
      (signal) => {
        void attempt;
        return search.trim()
          ? torrentSearch({ search, signal })
          : Promise.resolve([]);
      },
      [search, attempt],
    ),
  );
  if (torrents.loading)
    return (
      <p className="loading-message" role="status">
        Searching available downloads…
      </p>
    );
  if (torrents.error)
    return (
      <ErrorNotice
        error={torrents.error}
        retry={() => setAttempt((value) => value + 1)}
      />
    );
  if (!torrents.value.length)
    return (
      <EmptyState title="No downloads found" icon="search">
        <p>Try adjusting the title or author in your search.</p>
      </EmptyState>
    );
  const sorted = sortBySeeds
    ? [...torrents.value].sort((a, b) => (b.seeders ?? 0) - (a.seeders ?? 0))
    : torrents.value;
  return (
    <div className="torrent-results">
      <div className="section-toolbar">
        <span className="muted small">{sorted.length} available downloads</span>
        <label className="checkbox-label">
          <input
            type="checkbox"
            checked={sortBySeeds}
            onChange={(event) => setSortBySeeds(event.target.checked)}
          />
          Most seeders first
        </label>
      </div>
      {error != null && <ErrorNotice error={error} />}
      <ul className="download-results">
        {sorted.map((torrent) => {
          const downloading =
            queued.includes(torrent.id) ||
            active.some(
              (item) =>
                item.hash.toLowerCase() === torrent.info_hash?.toLowerCase(),
            );
          return (
            <li key={torrent.id}>
              <span className="file-icon">
                <Icon name="file" />
              </span>
              <div className="download-info">
                <h3>{torrent.title}</h3>
                <p>
                  {formatBytes(torrent.size)}
                  <span>·</span>
                  {torrent.tracker}
                  <span>·</span>
                  <span
                    className={
                      (torrent.seeders ?? 0) > 0 ? "positive-text" : ""
                    }
                  >
                    {torrent.seeders ?? 0} seeders
                  </span>
                </p>
              </div>
              <button
                className="button button-secondary button-small"
                disabled={downloading || pending !== undefined}
                onClick={async () => {
                  setPending(torrent.id);
                  setError(undefined);
                  try {
                    await onSelectTorrent(torrent);
                    setQueued((items) => [...items, torrent.id]);
                  } catch (error) {
                    setError(error);
                  } finally {
                    setPending(undefined);
                  }
                }}
              >
                <Icon name={downloading ? "check" : "download"} size={16} />
                {downloading
                  ? "Queued"
                  : pending === torrent.id
                    ? "Adding…"
                    : "Download"}
              </button>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
