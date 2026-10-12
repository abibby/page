import type { ReactNode } from "react";
import { Icon } from "./icon";

export function EmptyState({
  title,
  children,
  icon = "book",
}: {
  title: string;
  children: ReactNode;
  icon?: "book" | "search" | "download";
}) {
  return (
    <div className="empty-state">
      <span className="empty-icon">
        <Icon name={icon} size={30} />
      </span>
      <h2>{title}</h2>
      <div>{children}</div>
    </div>
  );
}

export function ErrorNotice({
  error,
  retry,
}: {
  error: unknown;
  retry?: () => void;
}) {
  return (
    <div className="error-notice" role="alert">
      <Icon name="alert" />
      <div>
        <strong>Something went wrong</strong>
        <p>
          {error instanceof Error
            ? error.message
            : "Unable to complete this request. Please try again."}
        </p>
      </div>
      {retry && (
        <button className="button button-secondary" onClick={retry}>
          Try again
        </button>
      )}
    </div>
  );
}

export function BookSkeletons() {
  return (
    <div className="book-grid" role="status" aria-label="Loading books">
      {Array.from({ length: 6 }, (_, i) => (
        <div className="book-skeleton" key={i}>
          <div />
          <span />
          <span />
        </div>
      ))}
    </div>
  );
}
