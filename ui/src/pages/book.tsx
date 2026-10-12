import { useState } from "react";
import { Link } from "react-router";
import { TorrentSearch } from "../components/search";
import { Cover } from "../components/book-card";
import { useRouteLoaderData } from "../routes";
import { Layout } from "../components/layout";
import { useDebounce } from "../hooks/use-debounce";
import { torrentAdd } from "../api/api";
import { UploadButton } from "../components/upload-button";
import { Icon } from "../components/icon";
import {
  fileExtension,
  fileName,
  formatBytes,
  formatDate,
  isAudio,
} from "../lib/format";

export function BookView() {
  const { book } = useRouteLoaderData<"book.view">();
  const [query, setQuery] = useState(`${book.title} ${book.authors}`);
  const search = useDebounce(query, 350);
  const [showDownloads, setShowDownloads] = useState(false);
  const formats = book.formats ?? [];
  const types = [...new Set(formats.map(fileExtension))];
  // Descriptions can contain publisher HTML. Render only its text, never executable markup.
  const description =
    new DOMParser().parseFromString(
      (book.description ?? "").replace(/<\/(p|div|li)>|<br\s*\/?\s*>/gi, "\n"),
      "text/html",
    ).body.textContent ?? "";
  return (
    <Layout>
      <Link to="/" className="back-link">
        <Icon name="back" size={17} />
        Back to library
      </Link>
      <section className="book-detail">
        <div className="detail-cover">
          <Cover title={book.title} author={book.authors} url={book.cover} />
          <span className="detail-cover-caption">
            FROM YOUR PERSONAL LIBRARY
          </span>
        </div>
        <div className="detail-info">
          <span className="eyebrow">
            {book.series
              ? `${book.series}${book.series_index ? ` · BOOK ${book.series_index}` : ""}`
              : "ON YOUR SHELF"}
          </span>
          <h1>{book.title}</h1>
          <Link
            className="author-link"
            to={`/?q=${encodeURIComponent(`authors:${book.authors}`)}`}
          >
            {book.authors || "Unknown author"}
          </Link>
          <div className="detail-badges">
            {types.map((type) => (
              <span className="format-badge" key={type}>
                {type}
              </span>
            ))}
            {!types.length && (
              <span className="format-badge">Metadata only</span>
            )}
          </div>
          <div className="detail-actions">
            <UploadButton bookId={book.id} />
            <button
              className="button button-secondary"
              onClick={() => setShowDownloads((value) => !value)}
              aria-expanded={showDownloads}
              aria-controls="find-downloads"
            >
              <Icon name="search" size={17} />
              {showDownloads ? "Hide downloads" : "Find a download"}
            </button>
          </div>
          <div className="book-description">
            <h2>About this book</h2>
            {description.trim() ? (
              description
                .split("\n")
                .filter(Boolean)
                .map((line, index) => <p key={index}>{line}</p>)
            ) : (
              <p className="muted">
                This book doesn’t have a description yet. Its files and details
                are below.
              </p>
            )}
          </div>
          {(book.tags ?? []).length > 0 && (
            <div className="tag-list">
              {book.tags.map((tag) => (
                <Link key={tag} to={`/?q=${encodeURIComponent(`tags:${tag}`)}`}>
                  {tag}
                </Link>
              ))}
            </div>
          )}
          <dl className="metadata-grid">
            <div>
              <dt>Published</dt>
              <dd>{formatDate(book.pubdate)}</dd>
            </div>
            <div>
              <dt>Language</dt>
              <dd>{(book.languages ?? []).join(", ") || "Not available"}</dd>
            </div>
            <div>
              <dt>Library size</dt>
              <dd>{formatBytes(book.size)}</dd>
            </div>
            {book.isbn && (
              <div>
                <dt>ISBN</dt>
                <dd>{book.isbn}</dd>
              </div>
            )}
          </dl>
        </div>
      </section>
      <section className="content-section">
        <div className="section-heading">
          <div>
            <span className="eyebrow">READY WHEN YOU ARE</span>
            <h2>
              Your files <span className="count-badge">{formats.length}</span>
            </h2>
          </div>
          <Icon name="file" size={23} />
        </div>
        {formats.length ? (
          <ul className="file-list">
            {formats.map((file) => (
              <li key={file}>
                <span className="file-icon">
                  <Icon
                    name={isAudio(file) ? "headphones" : "file"}
                    size={21}
                  />
                </span>
                <div className="file-info">
                  <strong>{fileName(file)}</strong>
                  <span>
                    {fileExtension(file)}
                    {isAudio(file) ? " · Audio" : " · Ebook"}
                  </span>
                  {isAudio(file) && (
                    <audio
                      controls
                      preload="none"
                      src={file}
                      aria-label={`Listen to ${fileName(file)}`}
                    />
                  )}
                </div>
                <a
                  className="button button-secondary button-small"
                  href={file}
                  download
                >
                  <Icon name="download" size={16} />
                  Download
                </a>
              </li>
            ))}
          </ul>
        ) : (
          <div className="no-files">
            <Icon name="upload" size={26} />
            <div>
              <h3>Your book is on the shelf. Add its first file.</h3>
              <p>Upload an ebook or audiobook using the button above.</p>
            </div>
          </div>
        )}
      </section>
      {showDownloads && (
        <section className="content-section" id="find-downloads">
          <div className="section-heading">
            <div>
              <span className="eyebrow">BUILD YOUR COLLECTION</span>
              <h2>Find a download</h2>
            </div>
          </div>
          <label className="search-field">
            <Icon name="search" size={19} />
            <input
              aria-label="Search torrents"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
            />
          </label>
          <p className="muted small">
            Search your configured indexers and send a download to your torrent
            client.
          </p>
          <TorrentSearch
            search={search}
            onSelectTorrent={async (torrent) => {
              const url = torrent.magnet_uri || torrent.link;
              if (!url) throw new Error("This result has no download link.");
              await torrentAdd({ url });
            }}
          />
        </section>
      )}
    </Layout>
  );
}
