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

export const torrentEvents = new EventTarget();

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

export type TorrentActiveRequest = BaseRequest & {};
export async function torrentActive(
  r: TorrentActiveRequest = {},
): Promise<ActiveTorrent[]> {
  return fetch("/api/torrent/active", {
    signal: r.signal,
  }).then((r) => r.json());
}

export type TorrentAddRequest = {
  url: string;
};
export async function torrentAdd(r: TorrentAddRequest): Promise<Torrent[]> {
  return fetch("/api/torrent", {
    method: "POST",
    body: JSON.stringify(r),
  })
    .then((r) => r.json())
    .finally(() => {
      torrentEvents.dispatchEvent(new Event("add"));
    });
}
