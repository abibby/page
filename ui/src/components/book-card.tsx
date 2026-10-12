import { useState } from "react";
import { Link } from "react-router";
import type { Book } from "../api/api";
import { fileExtension, isAudio } from "../lib/format";
import { Icon } from "./icon";

export function Cover({
  title,
  author,
  url,
}: {
  title: string;
  author: string;
  url?: string;
}) {
  const [failedURL, setFailedURL] = useState<string>();
  const hue = [...title].reduce((sum, char) => sum + char.charCodeAt(0), 0) % 5;
  return (
    <div className={`cover cover-color-${hue}`}>
      {url && failedURL !== url ? (
        <img
          src={url}
          alt=""
          loading="lazy"
          onError={() => setFailedURL(url)}
        />
      ) : (
        <div className="cover-fallback">
          <span className="cover-edition">THE PERSONAL LIBRARY</span>
          <span className="cover-ornament">✳</span>
          <strong>{title}</strong>
          <span className="cover-author">{author || "Unknown author"}</span>
          <Icon name="book" size={21} />
        </div>
      )}
    </div>
  );
}

export function BookCard({ book }: { book: Book }) {
  const formats = [...new Set((book.formats ?? []).map(fileExtension))];
  return (
    <Link className="book-card" to={`/book/${book.id}`}>
      <div className="book-cover-wrap">
        <Cover title={book.title} author={book.authors} url={book.cover} />
        <span className="cover-open">
          <Icon name="arrow" size={17} />
        </span>
      </div>
      <div className="book-card-info">
        <h3>{book.title}</h3>
        <p>{book.authors || "Unknown author"}</p>
        <div className="book-formats">
          <Icon
            name={(book.formats ?? []).some(isAudio) ? "headphones" : "book"}
            size={13}
          />
          <span>
            {formats.length
              ? formats.slice(0, 3).join(" · ")
              : "Awaiting a file"}
          </span>
        </div>
      </div>
      <span className="book-row-arrow">
        <Icon name="chevron" />
      </span>
    </Link>
  );
}
