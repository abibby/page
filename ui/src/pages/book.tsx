import { Book, TorrentSearch } from "../components/search";
import { useRouteLoaderData } from "../routes";
import { Layout } from "../components/layout";
import { useCallback, useMemo, useState, type ChangeEventHandler } from "react";
import { useDebounce } from "../hooks/use-debounce";
import { useUpdateCallback } from "../hooks/use-update-callback";
import { torrentAdd, type Torrent } from "../api/api";

export function BookView() {
  const data = useRouteLoaderData<"book.view">();
  const b = data.book;
  const [query, setQuery] = useState(b.title + " " + b.authors);
  const queryInputChange = useUpdateCallback(setQuery);

  const queryDebounce = useDebounce(query);

  const [showTorrents, setShowTorrents] = useState(false);

  const showTorrentsChange = useCallback<ChangeEventHandler<HTMLInputElement>>(
    (e) => {
      setShowTorrents(e.currentTarget.checked);
    },
    [],
  );
  const selectTorrent = useCallback(async (torrent: Torrent) => {
    await torrentAdd({ url: torrent.magnet_uri });
  }, []);

  const groupedFiles = useMemo(() => {
    const m = new Map<string, string[]>();
    for (const f of b.files) {
      const ext = f.match(/\.[^\.]+$/)?.[0] ?? "";
      let arr = m.get(ext);
      if (!arr) {
        arr = [];
        m.set(ext, arr);
      }
      arr.push(f);
    }
    return m;
  }, [b.files]);

  if (!b) {
    return <>404 not found</>;
  }

  return (
    <Layout>
      <Book title={b.title} author={b.authors} coverURL={b.cover} />

      {b.description.split("\n").map((line) => (
        <p key={line}>{line}</p>
      ))}

      <h3>Files</h3>
      <ul>
        {Array.from(groupedFiles.entries()).map(([ext, files]) => (
          <li key={ext}>
            {ext} {files.length}
          </li>
        ))}
      </ul>
      {/* <pre>{JSON.stringify(b, undefined, "    ")}</pre> */}

      <label>
        Search For Torrents
        <input
          type="checkbox"
          checked={showTorrents}
          onChange={showTorrentsChange}
        />
      </label>
      {showTorrents && (
        <>
          <input type="text" value={query} onInput={queryInputChange} />
          <TorrentSearch
            search={queryDebounce}
            onSelectTorrent={selectTorrent}
          />
        </>
      )}
    </Layout>
  );
}

export const Component = BookView;
