import { useCallback, useEffect, useRef, useState, type SubmitEventHandler } from "react";
import "./App.css";
import type { Book } from "./api/api";
import { list } from "./api/api";
import { Search } from "./components/search";

function App() {
  const [books, setBooks] = useState<Book[]>([]);
  const [query, setQuery] = useState("");
  const searchInputRef = useRef<HTMLInputElement | null>(null);

  useEffect(() => {
    list({ search: query, order: 'timestamp' }).then((b) => setBooks(b));
  }, [query]);

  const search = useCallback<SubmitEventHandler<HTMLFormElement>>((e) => {
    e.preventDefault();
    setQuery(searchInputRef.current?.value ?? "");
  }, []);

  return (
    <>

      <Search />

      
      <form onSubmit={search}>
        <input ref={searchInputRef} name="query" />
        <button type="submit">Search</button>
      </form>
      
      <form action="/api/book" method="post" encType="multipart/form-data">
        <input type="file" name="file" />
        <button type="submit">Add</button>
      </form>

      <ul className="book-list">
        {books.map((b) => (
          <li>
            <img src={b.cover} alt="" loading="lazy" />
            <div>{b.title}</div>
            <div>{b.authors}</div>
            <div>{b.series && `${b.series} [${b.series_index}]`}</div>
          </li>
        ))}
      </ul>
    </>
  );
}

export default App;
