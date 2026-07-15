import { useCallback, useRef, useState, type SubmitEventHandler } from "react";
import "./App.css";
import { bookImport, bookList } from "./api/api";
import { Book, BookList, Search } from "./components/search";
import { useAsync } from "./hooks/use-async";

function App() {
  const [query, setQuery] = useState("");
  const searchInputRef = useRef<HTMLInputElement | null>(null);
  const importInputRef = useRef<HTMLInputElement | null>(null);

  const books = useAsync(
    useCallback(
      (signal) =>
        bookList({ search: query, order: "timestamp", signal: signal }),
      [query],
    ),
  );

  const search = useCallback<SubmitEventHandler<HTMLFormElement>>((e) => {
    e.preventDefault();
    setQuery(searchInputRef.current?.value ?? "");
  }, []);

  const upload = useCallback<SubmitEventHandler<HTMLFormElement>>((e) => {
    e.preventDefault();

    if (!importInputRef.current?.files?.[0]) {
      return;
    }
    bookImport({
      file: importInputRef.current.files[0],
    });
  }, []);

  return (
    <>
      <h1>Page</h1>
      <h2>Find New Book</h2>
      <Search />

      <h2>Import Book</h2>
      <form onSubmit={upload}>
        <input ref={importInputRef} type="file" multiple name="file" />
        <button type="submit">Import Books</button>
      </form>

      <h2>Latest Books</h2>

      <form onSubmit={search}>
        <input ref={searchInputRef} name="query" />
        <button type="submit">Search</button>
      </form>
      <BookList>
        {books.value?.map((b) => (
          <li>
            <Book title={b.title} author={b.authors} coverURL={b.cover} />
          </li>
        ))}
      </BookList>
    </>
  );
}

export default App;
