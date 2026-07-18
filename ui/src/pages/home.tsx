import { useCallback, useState } from "react";
import { bookList } from "../api/api";
import { Book, BookList } from "../components/search";
import { useAsync } from "../hooks/use-async";
import { Link } from "react-router";
import { routePath } from "../routes";
import { useUpdateCallback } from "../hooks/use-update-callback";
import { useDebounce } from "../hooks/use-debounce";
import { Layout } from "../components/layout";

export function Home() {
  const [query, setQuery] = useState("");
  const searchInputChange = useUpdateCallback(setQuery);

  const queryDebounce = useDebounce(query);

  const books = useAsync(
    useCallback(
      (signal) =>
        bookList({ search: queryDebounce, order: "timestamp", signal: signal }),
      [queryDebounce],
    ),
  );

  return (
    <Layout>
      <Link to={routePath("book.search")}>Search</Link>
      <input value={query} onInput={searchInputChange} />
      <BookList>
        {books.value?.map((b) => (
          <li key={b.id}>
            <Link to={routePath("book.view", { bookId: b.id })}>
              <Book title={b.title} author={b.authors} coverURL={b.cover} />
            </Link>
          </li>
        ))}
      </BookList>
    </Layout>
  );
}

export const Component = Home;
