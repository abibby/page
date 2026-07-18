import { useSignals } from "@preact/signals-react/runtime";
import { torrentActive, torrentEvents, type ActiveTorrent } from "../api/api";
import { signal } from "@preact/signals-react";
import { useEffect } from "react";
let activeTorrents = signal<ActiveTorrent[]>([]);

let numActive = 0;

async function updateActiveTorrents() {
  const torrents = await torrentActive();
  activeTorrents.value = torrents;
}

torrentEvents.addEventListener("add", () => {
  updateActiveTorrents();
});
updateActiveTorrents();

setInterval(() => {
  if (numActive > 0) {
    updateActiveTorrents();
  }
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
