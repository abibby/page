import {
  useCallback,
  useRef,
  useState,
  type PropsWithChildren,
  type SubmitEventHandler,
} from "react";
import { useAsync } from "../hooks/use-async";
import {
  hardcoverSearch,
  torrentSearch,
  type Book,
  type HardcoverBook,
  type Torrent,
} from "../api/api";
import styles from "./search.module.css";

export function Search() {
  const hardcoverSearchRef = useRef<HTMLInputElement | null>(null);
  const torrentSearchRef = useRef<HTMLInputElement | null>(null);
  const [hardcoverSearch, setHardcoverSearch] = useState("");
  const [hardcoverBook, setHardcoverBook] = useState<HardcoverBook>();
  const [torrentSearch, setTorrentSearch] = useState("");
  const [torrent, setTorrent] = useState<Torrent>();

  const searchHardcoverSubmit = useCallback<
    SubmitEventHandler<HTMLFormElement>
  >((e) => {
    e.preventDefault();
    setHardcoverSearch(hardcoverSearchRef.current?.value ?? "");
    setHardcoverBook(undefined);
    setTorrent(undefined);
  }, []);
  const searchTorrentSubmit = useCallback<SubmitEventHandler<HTMLFormElement>>(
    (e) => {
      e.preventDefault();
      setTorrentSearch(torrentSearchRef.current?.value ?? "");
      setTorrent(undefined);
    },
    [],
  );

  const selectHardcoverBook = useCallback((b: HardcoverBook) => {
    setHardcoverBook(b);
    setTimeout(() => {
      if (torrentSearchRef.current) {
        torrentSearchRef.current.value = b.title + " " + hardcoverAuthor(b);
      }
    }, 1);
  }, []);

  return (
    <section>
      <form onSubmit={searchHardcoverSubmit}>
        <input
          ref={hardcoverSearchRef}
          type="text"
          placeholder="Search new books"
        />
        <button>Search</button>
      </form>

      {!hardcoverBook ? (
        <BookSearch
          search={hardcoverSearch}
          onSelectBook={selectHardcoverBook}
        />
      ) : (
        <HardcoverBook book={hardcoverBook} />
      )}

      <form onSubmit={searchTorrentSubmit}>
        <input
          ref={torrentSearchRef}
          type="text"
          placeholder="Search torrents"
        />
        <button>Search</button>
      </form>

      {!torrent ? (
        <TorrentSearch search={torrentSearch} onSelectTorrent={setTorrent} />
      ) : (
        <TorrentInfo torrent={torrent} />
      )}

      <pre>{JSON.stringify(torrent, undefined, "    ")}</pre>
    </section>
  );
}

type BookSearchParams = {
  search: string;
  onSelectBook: (b: HardcoverBook) => void;
};

function BookSearch(params: BookSearchParams) {
  const books = useAsync(
    useCallback(
      async (s: AbortSignal) => {
        if (params.search === "") {
          return [];
        }
        return hardcoverSearch({
          search: params.search,
          signal: s,
        });
      },
      [params.search],
    ),
  );
  if (books.loading) {
    return <div>Loading...</div>;
  }

  if (books.error) {
    return <div>{books.error.toString()}</div>;
  }

  return (
    <ul className={styles.bookList}>
      {books.value.map((b) => (
        <li key={b.id} onClick={() => params.onSelectBook(b)}>
          <HardcoverBook book={b} />
        </li>
      ))}
    </ul>
  );
}

type HardcoverBookProps = {
  book: HardcoverBook;
};

function HardcoverBook(props: HardcoverBookProps) {
  return (
    <Book
      title={props.book.title}
      coverURL={props.book.image.url}
      author={hardcoverAuthor(props.book)}
    />
  );
}

export type BookProps = {
  title: string;
  coverURL: string;
  author: string;
};

export function Book(props: BookProps) {
  return (
    <div className={styles.book}>
      <img src={props.coverURL} alt="" />
      <div>{props.title}</div>
      <div>{props.author}</div>
    </div>
  );
}

export function BookList(props: PropsWithChildren) {
  return <ul className={styles.bookList}>{props.children}</ul>;
}

function hardcoverAuthor(book: HardcoverBook): string {
  return book.contributions
    .map((a) => a.author?.name)
    .filter(Boolean)
    .join(" & ");
}

type TorrentSearchParams = {
  search: string;
  onSelectTorrent: (b: Torrent) => void;
};

function TorrentSearch(params: TorrentSearchParams) {
  const torrents = useAsync(
    useCallback(
      async (s: AbortSignal) => {
        console.log(params.search);
        if (params.search === "") {
          return [];
        }
        return torrentSearch({
          search: params.search,
          signal: s,
        });
      },
      [params.search],
    ),
  );

  if (torrents.loading) {
    return <div>Loading...</div>;
  }

  if (torrents.error) {
    return <div>{torrents.error.toString()}</div>;
  }

  if (torrents.value.length === 0) {
    return <div>No Results</div>;
  }

  return (
    <ul className={styles.torrentList}>
      <div className={`${styles.torrent} ${styles.torrentHeader}`}>
        <div>Title</div>
        <div>Size</div>
        <div>Seeders</div>
        <div>Peers</div>
        <div>Tracker</div>
      </div>
      {torrents.value.map((t) => (
        <li key={t.id} onClick={() => params.onSelectTorrent(t)}>
          <TorrentInfo torrent={t} />
        </li>
      ))}
    </ul>
  );
}

type TorrentInfoProps = {
  torrent: Torrent;
};

function TorrentInfo(props: TorrentInfoProps) {
  return (
    <div className={styles.torrent}>
      <div>{props.torrent.title}</div>
      <div>{formatBytes(props.torrent.size)}</div>
      <div>{props.torrent.seeders}</div>
      <div>{props.torrent.peers}</div>
      <div>{props.torrent.tracker}</div>
    </div>
  );
}

function formatBytes(bytes: number, decimals = 2) {
  if (bytes === 0) return "0 Bytes";

  const k = 1024;
  const dm = decimals < 0 ? 0 : decimals;
  const sizes = ["Bytes", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"];

  const i = Math.floor(Math.log(bytes) / Math.log(k));

  return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + " " + sizes[i];
}
