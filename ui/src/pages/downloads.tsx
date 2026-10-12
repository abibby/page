import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router";
import { torrentActive, updateEvents } from "../api/api";
import { useAsync } from "../hooks/use-async";
import { Layout } from "../components/layout";
import { EmptyState, ErrorNotice } from "../components/feedback";
import { Icon } from "../components/icon";
import { formatBytes } from "../lib/format";

export function Downloads() {
  const [refresh, setRefresh] = useState(0);
  const data = useAsync(
    useCallback(
      (signal) => {
        void refresh;
        return torrentActive({ signal });
      },
      [refresh],
    ),
  );
  useEffect(() => {
    const update = () => setRefresh((value) => value + 1);
    const interval = setInterval(update, 10000);
    updateEvents.addEventListener("torrent", update);
    return () => {
      clearInterval(interval);
      updateEvents.removeEventListener("torrent", update);
    };
  }, []);
  return (
    <Layout>
      <div className="page-heading">
        <div>
          <span className="eyebrow">ON THE WAY TO YOUR SHELF</span>
          <h1>
            Downloads<span className="title-dot">.</span>
          </h1>
          <p>A little patience. A whole new story.</p>
        </div>
        <button
          className="button button-secondary"
          onClick={() => setRefresh((value) => value + 1)}
          disabled={data.loading}
        >
          Refresh
        </button>
      </div>
      <div className="section-toolbar downloads-toolbar">
        <h2>Download activity</h2>
        <span className="muted small">Updates every 10 seconds</span>
      </div>
      {data.loading ? (
        <p role="status" className="loading-message">
          Checking your downloads…
        </p>
      ) : data.error ? (
        <ErrorNotice
          error={data.error}
          retry={() => setRefresh((value) => value + 1)}
        />
      ) : !data.value.length ? (
        <EmptyState title="All quiet on this shelf" icon="download">
          <p>
            No active downloads. Find a book, then search for a download from
            its details page.
          </p>
          <Link className="button button-secondary" to="/">
            Browse your library <Icon name="arrow" size={16} />
          </Link>
        </EmptyState>
      ) : (
        <ul className="activity-list">
          {data.value.map((torrent) => {
            const progress = Math.round(
              Math.min(1, Math.max(0, torrent.progress || 0)) * 100,
            );
            return (
              <li key={torrent.hash}>
                <span className="file-icon">
                  <Icon
                    name={progress === 100 ? "check" : "download"}
                    size={23}
                  />
                </span>
                <div className="activity-info">
                  <div className="activity-heading">
                    <h3>{torrent.name}</h3>
                    <span>{progress}%</span>
                  </div>
                  <progress
                    max={100}
                    value={progress}
                    aria-label={`${torrent.name} progress`}
                  />
                  <div className="activity-meta">
                    <span>
                      {torrent.state.replace(/([a-z])([A-Z])/g, "$1 $2")}
                    </span>
                    <span>
                      {formatBytes(torrent.downloaded)} /{" "}
                      {formatBytes(torrent.total_size || torrent.size)}
                    </span>
                    <span>{formatBytes(torrent.dlspeed)}/s</span>
                  </div>
                </div>
              </li>
            );
          })}
        </ul>
      )}
    </Layout>
  );
}
