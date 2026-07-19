import { useSignals } from "@preact/signals-react/runtime";
import { torrentActive, updateEvents, type ActiveTorrent } from "../api/api";
import { signal } from "@preact/signals-react";
import { useEffect } from "react";
let activeTorrents = signal<ActiveTorrent[]>([]);

let numActive = 0;

async function updateActiveTorrents() {
  if (numActive == 0) {
    return;
  }
  const torrents = await torrentActive();
  activeTorrents.value = torrents;
}

updateEvents.addEventListener("torrent", () => {
  updateActiveTorrents();
});
updateActiveTorrents();

setInterval(() => {
  updateActiveTorrents();
}, 10_000);

export function useActiveTorrents(): ActiveTorrent[] {
  useSignals();

  useEffect(() => {
    numActive++;
    return () => {
      numActive--;
    };
  }, []);

  return activeTorrents.value;
}
