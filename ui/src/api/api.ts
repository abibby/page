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

export type FullBook = Book & {
  description: string;
  files: string[];
};

export type HardcoverBook = {
  id: string | number;
  title: string;
  image: { url: string } | null;
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
  peers?: number;
  seeders?: number;
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

export type ActiveTorrent = {
  added_on: number;
  amount_left: number;
  auto_tmm: boolean;
  availability: number;
  category: string;
  comment: string;
  completed: number;
  completion_on: number;
  created_by: string;
  content_path: string;
  dl_limit: number;
  dlspeed: number;
  download_path: string;
  downloaded: number;
  downloaded_session: number;
  eta: number;
  f_l_piece_prio: boolean;
  force_start: boolean;
  hash: string;
  infohash_v1: string;
  infohash_v2: string;
  popularity: number;
  private: boolean;
  last_activity: number;
  magnet_uri: string;
  max_ratio: number;
  max_seeding_time: number;
  max_inactive_seeding_time: number;
  name: string;
  num_complete: number;
  num_incomplete: number;
  num_leechs: number;
  num_seeds: number;
  priority: number;
  progress: number;
  ratio: number;
  ratio_limit: number;
  reannounce: number;
  save_path: string;
  seeding_time: number;
  seeding_time_limit: number;
  inactive_seeding_time_limit: number;
  share_limit_action: string;
  share_limits_mode: string;
  seen_complete: number;
  seq_dl: boolean;
  size: number;
  state: string;
  super_seeding: boolean;
  tags: string;
  time_active: number;
  total_size: number;
  tracker: string;
  trackers_count: number;
  up_limit: number;
  uploaded: number;
  uploaded_session: number;
  upspeed: number;
  trackers: unknown;
};

async function readResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    if (response.status === 404)
      throw new Response("Not found", { status: 404 });
    let message = `Request failed (${response.status}). Please try again.`;
    try {
      const body = await response.json();
      if (typeof body.message === "string") message = body.message;
      else if (typeof body.error === "string") message = body.error;
    } catch {
      /* The server may return a non-JSON error. */
    }
    throw new Error(message);
  }
  if (response.status === 204) return undefined as T;
  return response.json();
}

async function readList<T>(response: Response): Promise<T[]> {
  return (await readResponse<T[] | null>(response)) ?? [];
}

export const updateEvents = new EventTarget();

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
  p.set("limit", "99");
  if (r.order) {
    p.set("order", r.order);
  }
  if (r.ascending) {
    p.set("ascending", "true");
  }
  return fetch("/api/book?" + p.toString(), {
    signal: r.signal,
  }).then(readList<Book>);
}

export type BookViewRequest = BaseRequest & {};

export async function bookView(
  id: number,
  r: BookViewRequest = {},
): Promise<FullBook> {
  return fetch(`/api/book/${id}`, {
    signal: r.signal,
  }).then(readResponse<FullBook>);
}

export type BookImportRequest = BaseRequest & {
  file: File;
  book_id?: number;
};
export type BookImportResponse = {
  book_id: number;
};

export async function bookImport(
  r: BookImportRequest,
): Promise<BookImportResponse> {
  const data = new FormData();
  data.append("file", r.file);
  if (r.book_id) {
    data.append("book_id", String(r.book_id));
  }
  return fetch("/api/book/import", {
    method: "POST",
    body: data,
    signal: r.signal,
  })
    .then(readResponse<BookImportResponse>)
    .then((result) => {
      updateEvents.dispatchEvent(new Event("book"));
      return result;
    });
}

export type BookAddRequest = {
  title: string;
  authors: string[];
  hardcover_id: number;
};
export type BookAddResponse = {
  book_id: number;
};
export async function bookAdd(r: BookAddRequest): Promise<BookAddResponse> {
  return fetch("/api/book", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(r),
  })
    .then(readResponse<BookAddResponse>)
    .then((result) => {
      updateEvents.dispatchEvent(new Event("book"));
      return result;
    });
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
  }).then(readList<HardcoverBook>);
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
  }).then(readList<Torrent>);
}

export type TorrentActiveRequest = BaseRequest & {};
export async function torrentActive(
  r: TorrentActiveRequest = {},
): Promise<ActiveTorrent[]> {
  return fetch("/api/torrent/active", {
    signal: r.signal,
  }).then(readList<ActiveTorrent>);
}

export type TorrentAddRequest = {
  url: string;
};
export async function torrentAdd(r: TorrentAddRequest): Promise<void> {
  const response = await fetch("/api/torrent", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(r),
  });
  if (!response.ok) await readResponse(response);
  updateEvents.dispatchEvent(new Event("torrent"));
}
