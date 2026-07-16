import { useSignals } from "@preact/signals-react/runtime";
import { torrentActive, torrentEvents, type ActiveTorrent } from "../api/api";
import { signal } from "@preact/signals-react";
let activeTorrents = signal<ActiveTorrent[]>([]);

async function updateActiveTorrents() {
  const torrents = await torrentActive();
  activeTorrents.value = torrents;
}

torrentEvents.addEventListener("add", () => {
  updateActiveTorrents();
});
updateActiveTorrents();

export function useActiveTorrents(): ActiveTorrent[] {
  useSignals();
  return activeTorrents.value;
}
