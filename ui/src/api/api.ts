export type Book = {
  id: number;
  title: string;
  authors: string;
  author_sort: string;
  formats: string[];
  identifiers: Record<string, string>;
  cover: string;
  isbn: string;
  languages: string[];
  last_modified: string;
  pubdate: string;
  series: string;
  series_index: number;
  size: number;
  tags: string[];
  timestamp: string;
  uuid: string;
};

export type HardcoverBook = {
  id: number;
  title: string;
  image: { url: string };
  contributions: Contribution[];
};

export type Contribution = {
  author?: Contributor;
};
export type Contributor = {
  name: string;
};

export type Torrent = {
  id: string;
  info_hash: string;
  tracker: string;
  tracker_id: string;
  tracker_type: string;
  grabs: number;
  peers: number;
  seeders: number;
  upload_volume_factor: number;
  title: string;
  categories: number[];
  cover_url?: string;
  link: string;
  magnet_uri: string;
  size: number;
  publish_date: string;
  languages: string[];
  subs: string[];
  genres: string[];
  tracks: string[];
};

export type Field =
  | "author_sort"
  | "authors"
  | "comments"
  | "cover"
  | "formats"
  | "identifiers"
  | "isbn"
  | "languages"
  | "last_modified"
  | "pubdate"
  | "publisher"
  | "rating"
  | "series"
  | "series_index"
  | "size"
  | "tags"
  | "template"
  | "timestamp"
  | "title"
  | "uuid"
  | "all";

export type BaseRequest = {
  signal?: AbortSignal;
};

export type BookListRequest = BaseRequest & {
  search?: string;
  order?: Field;
  ascending?: boolean;
};

export async function bookList(r: BookListRequest): Promise<Book[]> {
  const p = new URLSearchParams();
  p.set("search", r.search ?? "");
  if (r.order) {
    p.set("order", r.order);
  }
  if (r.ascending) {
    p.set("ascending", "true");
  }
  return fetch("/api/book?" + p.toString(), {
    signal: r.signal,
  }).then((r) => r.json());
}

export type BookImportRequest = BaseRequest & {
  file: File;
  hardcover_id?: Number;
};

export async function bookImport(r: BookImportRequest): Promise<Book[]> {
  var data = new FormData();
  data.append("file", r.file);
  if (r.hardcover_id) {
    data.append("hardcover_id", String(r.hardcover_id));
  }
  return fetch("/api/book", {
    method: "POST",
    body: data,
  }).then((r) => r.json());
}

export type HardcoverSearchRequest = BaseRequest & {
  search: string;
};
export async function hardcoverSearch(
  r: HardcoverSearchRequest,
): Promise<HardcoverBook[]> {
  const p = new URLSearchParams();
  p.set("q", r.search);
  return fetch("/api/hardcover/search?" + p.toString(), {
    signal: r.signal,
  }).then((r) => r.json());
}

export type TorrentSearchRequest = BaseRequest & {
  search: string;
};
export async function torrentSearch(
  r: TorrentSearchRequest,
): Promise<Torrent[]> {
  const p = new URLSearchParams();
  p.set("q", r.search);
  return fetch("/api/torrent/search?" + p.toString(), {
    signal: r.signal,
  }).then((r) => r.json());
}
