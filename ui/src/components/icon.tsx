import type { CSSProperties } from "react";

const paths = {
  book: (
    <>
      <path d="M12 7c-3-2-6-2-9-1v14c3-1 6-1 9 1 3-2 6-2 9-1V6c-3-1-6-1-9 1Z" />
      <path d="M12 7v14" />
    </>
  ),
  library: (
    <>
      <path d="M4 4v16M8 4v16M12 4v16M16 4l4 16M2 20h20" />
    </>
  ),
  search: (
    <>
      <circle cx="10.5" cy="10.5" r="6.5" />
      <path d="m16 16 5 5" />
    </>
  ),
  compass: (
    <>
      <circle cx="12" cy="12" r="9" />
      <path d="m16 8-2.5 5.5L8 16l2.5-5.5Z" />
    </>
  ),
  download: (
    <>
      <path d="M12 3v12m-4-4 4 4 4-4M4 16v5h16v-5" />
    </>
  ),
  upload: (
    <>
      <path d="M12 16V3m-4 4 4-4 4 4M4 16v5h16v-5" />
    </>
  ),
  plus: <path d="M12 5v14M5 12h14" />,
  grid: (
    <>
      <rect x="3" y="3" width="6" height="6" rx="1" />
      <rect x="15" y="3" width="6" height="6" rx="1" />
      <rect x="3" y="15" width="6" height="6" rx="1" />
      <rect x="15" y="15" width="6" height="6" rx="1" />
    </>
  ),
  list: (
    <>
      <path d="M9 5h12M9 12h12M9 19h12M3 5h1M3 12h1M3 19h1" />
    </>
  ),
  arrow: <path d="M4 12h16m-6-6 6 6-6 6" />,
  back: <path d="M20 12H4m6-6-6 6 6 6" />,
  headphones: (
    <>
      <path d="M4 15v-3a8 8 0 0 1 16 0v3" />
      <rect x="3" y="13" width="4" height="8" rx="2" />
      <rect x="17" y="13" width="4" height="8" rx="2" />
    </>
  ),
  close: <path d="m6 6 12 12M6 18 18 6" />,
  check: <path d="m5 12 4 4L19 6" />,
  file: (
    <>
      <path d="M14 3H5v18h14V8Zm0 0v5h5M8 13h8M8 17h5" />
    </>
  ),
  chevron: <path d="m9 5 7 7-7 7" />,
  alert: (
    <>
      <circle cx="12" cy="12" r="9" />
      <path d="M12 7v6m0 4h.01" />
    </>
  ),
} as const;

export function Icon({
  name,
  size = 20,
  style,
}: {
  name: keyof typeof paths;
  size?: number;
  style?: CSSProperties;
}) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.65"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      style={style}
    >
      {paths[name]}
    </svg>
  );
}
