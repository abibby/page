export function fileExtension(path: string) {
  return (
    path.split(/[?#]/)[0].match(/\.([^./]+)$/)?.[1] ?? "file"
  ).toUpperCase();
}

export function isAudio(path: string) {
  return [
    "MP3",
    "M4B",
    "M4A",
    "AAC",
    "FLAC",
    "OGG",
    "OPUS",
    "WAV",
    "WMA",
  ].includes(fileExtension(path));
}

export function formatBytes(bytes: number) {
  if (!Number.isFinite(bytes) || bytes <= 0) return "0 B";
  const index = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), 4);
  return `${(bytes / 1024 ** index).toFixed(index ? 1 : 0)} ${["B", "KB", "MB", "GB", "TB"][index]}`;
}

export function fileName(path: string) {
  const name = path.split("/").pop() ?? path;
  try {
    return decodeURIComponent(name);
  } catch {
    return name;
  }
}

export function formatDate(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) || date.getFullYear() < 1000
    ? "Not available"
    : date.toLocaleDateString(undefined, {
        year: "numeric",
        month: "short",
        day: "numeric",
      });
}
