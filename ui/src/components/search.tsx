import {
  useCallback,
  useMemo,
  useState,
  type ChangeEventHandler,
  type PropsWithChildren,
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
import { useActiveTorrents } from "../hooks/use-active-torrents";

export type BookSearchParams = {
  search: string;
  onSelectBook: (b: HardcoverBook) => void;
};

export function BookSearch(params: BookSearchParams) {
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

export type TorrentSearchParams = {
  search: string;
  onSelectTorrent: (b: Torrent) => void;
};

export function TorrentSearch(params: TorrentSearchParams) {
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
      <label>
        sort by seeds
        <input type="checkbox" checked={seedSort} onChange={seedSortInput} />
      </label>
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
        {activeTorrent && (
          <>
            {activeTorrent.state}{" "}
            {(
              Math.min(1, activeTorrent.downloaded / activeTorrent.total_size) *
              100
            ).toFixed(0)}
            %
          </>
        )}
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
