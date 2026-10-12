import { useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router";
import { BookSearch } from "../components/search";
import { useDebounce } from "../hooks/use-debounce";
import { Layout } from "../components/layout";
import { bookAdd, type HardcoverBook } from "../api/api";
import { useModal } from "../components/modal";
import { Prompt } from "../modals/prompt";
import { Icon } from "../components/icon";
import { ErrorNotice } from "../components/feedback";
import { UploadButton } from "../components/upload-button";

export function Search() {
  const modals = useModal();
  const navigate = useNavigate();
  const [params, setParams] = useSearchParams();
  const query = params.get("q") ?? "";
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>();
  const search = useDebounce(query, 350);
  const selectBook = async (book: HardcoverBook) => {
    if (!(await modals.openModal(Prompt, { message: book.title }))) return;
    setBusy(true);
    setError(undefined);
    try {
      const response = await bookAdd({
        title: book.title,
        authors: (book.contributions ?? []).flatMap((item) =>
          item.author ? [item.author.name] : [],
        ),
        hardcover_id: Number(book.id),
      });
      await navigate(`/book/${response.book_id}`);
    } catch (error) {
      setError(error);
    } finally {
      setBusy(false);
    }
  };
  return (
    <Layout>
      <div className="page-heading">
        <div>
          <span className="eyebrow">YOUR NEXT GREAT READ</span>
          <h1>
            Discover<span className="title-dot">.</span>
          </h1>
          <p>A new perspective is only a page away.</p>
        </div>
        <UploadButton />
      </div>
      <section className="discover-search">
        <span className="eyebrow">A WORLD OF STORIES</span>
        <h2>Find something you’ll get lost in.</h2>
        <label className="search-field search-field-large">
          <Icon name="search" size={22} />
          <input
            aria-label="Search Hardcover"
            placeholder="Search for a book or author…"
            value={query}
            onChange={(event) =>
              setParams(event.target.value ? { q: event.target.value } : {}, {
                replace: true,
              })
            }
            autoComplete="off"
          />
          {query && (
            <button
              className="icon-button"
              aria-label="Clear search"
              onClick={() => setParams({}, { replace: true })}
            >
              <Icon name="close" size={18} />
            </button>
          )}
        </label>
        <p>
          Book information from Hardcover. Already have a file? Import it
          directly.
        </p>
      </section>
      {error != null && <ErrorNotice error={error} />}
      {busy && (
        <p role="status" className="loading-message">
          Adding this book to your library…
        </p>
      )}
      <BookSearch search={search} onSelectBook={selectBook} busy={busy} />
      <div className="discover-footer">
        <Icon name="library" size={18} />
        <span>Your favorites deserve a place on the shelf.</span>
        <Link to="/">
          Back to your library <Icon name="arrow" size={15} />
        </Link>
      </div>
    </Layout>
  );
}
