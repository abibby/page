import {
  useCallback,
  useMemo,
  useRef,
  useState,
  type ChangeEventHandler,
  type PropsWithChildren,
  type SubmitEventHandler,
} from "react";
import { useAsync } from "../hooks/use-async";
import {
  hardcoverSearch,
  torrentAdd,
  torrentSearch,
  type Book,
  type HardcoverBook,
  type Torrent,
} from "../api/api";
import styles from "./search.module.css";
import { useActiveTorrents } from "../hooks/use-active-torrents";
import { useQueryString } from "../hooks/use-query-string";

export function Search() {
  const hardcoverSearchRef = useRef<HTMLInputElement | null>(null);
  const torrentSearchRef = useRef<HTMLInputElement | null>(null);

  const [hardcoverSearch, setHardcoverSearch] = useQueryString("book");
  const [torrentSearch, setTorrentSearch] = useQueryString("torrent");

  const [hardcoverBook, setHardcoverBook] = useState<HardcoverBook>();
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
      setHardcoverSearch("");
    },
    [],
  );

  const selectHardcoverBook = useCallback((b: HardcoverBook) => {
    setHardcoverBook(b);
    setTimeout(() => {
      if (torrentSearchRef.current) {
        torrentSearchRef.current.value = b.title + " " + hardcoverAuthor(b);
        setTorrentSearch(torrentSearchRef.current.value);
      }
    }, 1);
  }, []);

  const selectTorrent = useCallback(async (torrent: Torrent) => {
    setTorrent(torrent);
    await torrentAdd({ url: torrent.magnet_uri });
  }, []);

  return (
    <section>
      <form onSubmit={searchHardcoverSubmit}>
        <input
          ref={hardcoverSearchRef}
          type="text"
          placeholder="Search new books"
          defaultValue={hardcoverSearch}
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
          defaultValue={torrentSearch}
        />
        <button>Search</button>
      </form>

      {!torrent ? (
        <TorrentSearch search={torrentSearch} onSelectTorrent={selectTorrent} />
      ) : (
        <TorrentInfo torrent={torrent} />
      )}
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

  const [seedSort, setSeedSort] = useState(false);

  const seedSortInput = useCallback<ChangeEventHandler<HTMLInputElement>>(
    (e) => {
      setSeedSort(e.currentTarget.checked);
    },
    [],
  );

  const sortedList = useMemo(() => {
    if (seedSort) {
      return Array.from(torrents.value ?? []).sort(
        (a, b) => (b.seeders ?? 0) - (a.seeders ?? 0),
      );
    }
    return torrents.value ?? [];
  }, [torrents.value, seedSort]);

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
    <section>
      <input type="checkbox" checked={seedSort} onChange={seedSortInput} />
      <ul className={styles.torrentList}>
        <li>
          <div className={`${styles.torrent} ${styles.torrentHeader}`}>
            <div className={styles.title}>Title</div>
            <div className={styles.size}>Size</div>
            <div className={styles.peers}>Peers</div>
            <div className={styles.tracker}>Tracker</div>
          </div>
        </li>
        {sortedList.map((t) => (
          <li key={t.id} onClick={() => params.onSelectTorrent(t)}>
            <TorrentInfo torrent={t} />
          </li>
        ))}
      </ul>
    </section>
  );
}

type TorrentInfoProps = {
  torrent: Torrent;
};

function TorrentInfo(props: TorrentInfoProps) {
  const activeTorrents = useActiveTorrents();
  const activeTorrent = useMemo(() => {
    return activeTorrents.find(
      (t) => t.hash.toLowerCase() == props.torrent.info_hash?.toLowerCase(),
    );
  }, [activeTorrents, props.torrent.info_hash]);
  return (
    <div className={`${styles.torrent} ${activeTorrent && styles.active}`}>
      <div className={styles.title}>{props.torrent.title}</div>
      <div className={styles.size}>{formatBytes(props.torrent.size)}</div>
      <div className={styles.peers}>
        <span className={pillStyle(props.torrent.seeders ?? 0)}>
          {props.torrent.seeders ?? 0} / {props.torrent.peers ?? 0}
        </span>
      </div>
      <div className={styles.tracker}>{props.torrent.tracker}</div>
      <div className={styles.downloading}>
        {activeTorrent && activeTorrent.state}
      </div>
    </div>
  );
}

function pillStyle(seeders: number): string {
  let className = styles.pill + " ";
  if (seeders >= 15) {
    return className + styles.good;
  }
  if (seeders > 0) {
    return className + styles.ok;
  }
  return className + styles.bad;
}

function formatBytes(bytes: number, decimals = 2) {
  if (bytes === 0) return "0 Bytes";

  const k = 1024;
  const dm = decimals < 0 ? 0 : decimals;
  const sizes = ["Bytes", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"];

  const i = Math.floor(Math.log(bytes) / Math.log(k));

  return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + " " + sizes[i];
}
