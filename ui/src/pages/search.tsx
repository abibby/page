import { useCallback } from "react";
import { BookSearch } from "../components/search";
import { useDebounce } from "../hooks/use-debounce";
import { useQueryString } from "../hooks/use-query-string";
import { Layout } from "../components/layout";
import { useUpdateCallback } from "../hooks/use-update-callback";
import { bookAdd, type HardcoverBook } from "../api/api";
import { useModal } from "../components/modal";
import { Prompt } from "../modals/prompt";
import { useNavigate } from "react-router";
import { routePath } from "../routes";

export function Search() {
  const modals = useModal();
  const navigate = useNavigate();
  const [query, setQuery] = useQueryString("q");

  const queryDebounce = useDebounce(query);
  const queryInputChange = useUpdateCallback(setQuery);

  const selectBook = useCallback(async (book: HardcoverBook) => {
    const addBook = await modals.openModal(Prompt, { message: book.title });
    if (!addBook) {
      return;
    }
    const resp = await bookAdd({
      title: book.title,
      authors: book.contributions
        .map((c) => c.author?.name)
        .filter((n) => n !== undefined),
      hardcover_id: Number(book.id),
    });
    navigate(routePath("book.view", { bookId: resp.book_id }));
  }, []);

  return (
    <Layout>
      <input
        type="text"
        placeholder="Search new books"
        value={query}
        onInput={queryInputChange}
      />

      <BookSearch search={queryDebounce} onSelectBook={selectBook} />
    </Layout>
  );
}

export const Component = Search;
