import { useCallback, useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router";
import { bookList, updateEvents, type Field } from "../api/api";
import { useAsync } from "../hooks/use-async";
import { useDebounce } from "../hooks/use-debounce";
import { Layout } from "../components/layout";
import { UploadButton } from "../components/upload-button";
import { BookCard } from "../components/book-card";
import { BookSkeletons, EmptyState, ErrorNotice } from "../components/feedback";
import { Icon } from "../components/icon";
import { isAudio } from "../lib/format";

export function Home() {
  const [params, setParams] = useSearchParams();
  const query = params.get("q") ?? "";
  const type = params.get("type") ?? "all";
  const sort = params.get("sort") ?? "recent";
  const [view, setView] = useState(() => {
    try {
      return localStorage.getItem("page-view") === "list" ? "list" : "grid";
    } catch {
      return "grid";
    }
  });
  const [refreshKey, setRefreshKey] = useState(0);
  const debouncedQuery = useDebounce(query, 250);
  const changeParam = (key: string, value: string) =>
    setParams(
      (previous) => {
        const next = new URLSearchParams(previous);
        if (value) next.set(key, value);
        else next.delete(key);
        return next;
      },
      { replace: true },
    );
  const books = useAsync(
    useCallback(
      (signal) => {
        void refreshKey;
        const order: Field =
          sort === "title"
            ? "title"
            : sort === "author"
              ? "author_sort"
              : "timestamp";
        return bookList({
          search: debouncedQuery,
          order,
          ascending: sort !== "recent",
          signal,
        });
      },
      [debouncedQuery, sort, refreshKey],
    ),
  );
  useEffect(() => {
    const update = () => setRefreshKey((key) => key + 1);
    updateEvents.addEventListener("book", update);
    return () => updateEvents.removeEventListener("book", update);
  }, []);
  const filtered = (books.value ?? []).filter((book) =>
    type === "audio"
      ? (book.formats ?? []).some(isAudio)
      : type === "ebooks"
        ? (book.formats ?? []).some((format) => !isAudio(format))
        : true,
  );
  const title =
    type === "audio"
      ? "Your audiobooks"
      : type === "ebooks"
        ? "Your ebooks"
        : "Your library";
  return (
    <Layout>
      <div className="page-heading">
        <div>
          <span className="eyebrow">THE READING ROOM</span>
          <h1>
            {title}
            <span className="title-dot">.</span>
          </h1>
          <p>A place for every book. A book for every mood.</p>
        </div>
        <UploadButton />
      </div>
      <section className="library-banner" aria-label="Discover books">
        <div className="banner-copy">
          <span className="eyebrow">MAKE ROOM FOR A NEW CHAPTER</span>
          <h2>
            There’s a whole world
            <br />
            on your shelf.
          </h2>
          <p>Keep the stories you love, and find your next great read.</p>
          <Link to="/book/search">
            Explore new books <Icon name="arrow" size={17} />
          </Link>
        </div>
        <div className="shelf-art" aria-hidden="true">
          <div className="art-orbit" />
          <span className="art-star star-one">✳</span>
          <span className="art-star star-two">✧</span>
          <div className="art-book art-book-one">
            <span>
              THE ART OF
              <br />
              GETTING LOST
            </span>
            <i>VOLUME I</i>
          </div>
          <div className="art-book art-book-two">
            <span>
              ONE MORE
              <br />
              <em>chapter.</em>
            </span>
            <Icon name="book" size={34} />
          </div>
          <div className="art-book art-book-three">
            <span>A WORLD WITHIN</span>
          </div>
          <div className="art-shelf" />
        </div>
      </section>
      <div className="collection-heading">
        <div>
          <h2>{query ? "Search results" : "On your shelf"}</h2>
          {!books.loading && !books.error && (
            <span className="count-badge">{filtered.length}</span>
          )}
        </div>
        <span className="muted small">A collection that’s entirely you.</span>
      </div>
      <div className="library-toolbar">
        <label className="search-field">
          <Icon name="search" size={19} />
          <input
            aria-label="Search your library"
            placeholder="Search by title, author, or keyword…"
            value={query}
            onChange={(event) => changeParam("q", event.target.value)}
          />
          {query && (
            <button
              className="icon-button"
              aria-label="Clear search"
              onClick={() => changeParam("q", "")}
            >
              <Icon name="close" size={16} />
            </button>
          )}
        </label>
        <div className="toolbar-options">
          <label className="sort-control">
            <span>Sort by</span>
            <select
              aria-label="Sort books"
              value={sort}
              onChange={(event) => changeParam("sort", event.target.value)}
            >
              <option value="recent">Recently added</option>
              <option value="title">Title: A–Z</option>
              <option value="author">Author: A–Z</option>
            </select>
          </label>
          <div className="view-switch" aria-label="Book layout">
            {(["grid", "list"] as const).map((mode) => (
              <button
                key={mode}
                className={view === mode ? "selected" : ""}
                aria-label={`${mode === "grid" ? "Grid" : "List"} view`}
                aria-pressed={view === mode}
                onClick={() => {
                  setView(mode);
                  try {
                    localStorage.setItem("page-view", mode);
                  } catch {
                    /* Storage is optional. */
                  }
                }}
              >
                <Icon name={mode} size={17} />
              </button>
            ))}
          </div>
        </div>
      </div>
      <div className="filter-row">
        <div className="filter-tabs" aria-label="Filter by format">
          {[
            ["all", "All books"],
            ["ebooks", "Ebooks"],
            ["audio", "Audiobooks"],
          ].map(([key, label]) => (
            <button
              key={key}
              className={type === key ? "active" : ""}
              aria-pressed={type === key}
              onClick={() => changeParam("type", key === "all" ? "" : key)}
            >
              {label}
            </button>
          ))}
        </div>
        <span className="muted small">
          {books.loading
            ? "Finding your books…"
            : `${filtered.length} ${filtered.length === 1 ? "book" : "books"} shown`}
        </span>
      </div>
      {books.loading ? (
        <BookSkeletons />
      ) : books.error ? (
        <ErrorNotice
          error={books.error}
          retry={() => setRefreshKey((key) => key + 1)}
        />
      ) : filtered.length ? (
        <ul className={view === "grid" ? "book-grid" : "book-list"}>
          {filtered.map((book) => (
            <li key={book.id}>
              <BookCard book={book} />
            </li>
          ))}
        </ul>
      ) : (
        <EmptyState
          title={
            query
              ? "No books found"
              : type !== "all"
                ? "No books in this collection yet"
                : "Every library starts with a book"
          }
          icon={query ? "search" : "book"}
        >
          <p>
            {query
              ? "Try another title or author, or clear your search."
              : "Import a book you own or discover something new to add to your shelf."}
          </p>
          {query ? (
            <button
              className="button button-secondary"
              onClick={() => changeParam("q", "")}
            >
              Clear search
            </button>
          ) : (
            <Link className="button button-secondary" to="/book/search">
              Discover books <Icon name="arrow" size={16} />
            </Link>
          )}
        </EmptyState>
      )}
      {(books.value?.length ?? 0) >= 99 && (
        <p className="result-note">
          Showing up to 99 books from your library. Search to find more titles;
          format filters apply to these results.
        </p>
      )}
    </Layout>
  );
}
