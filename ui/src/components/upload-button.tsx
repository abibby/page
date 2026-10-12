import { useRef, useState } from "react";
import { bookImport } from "../api/api";
import { Link, useRevalidator } from "react-router";
import { Icon } from "./icon";

export function UploadButton({ bookId }: { bookId?: number }) {
  const input = useRef<HTMLInputElement>(null);
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<number>();
  const [error, setError] = useState("");
  const revalidator = useRevalidator();
  return (
    <div className="upload-control">
      <input
        ref={input}
        type="file"
        hidden
        onChange={async (event) => {
          const file = event.target.files?.[0];
          if (!file) return;
          setLoading(true);
          setError("");
          setResult(undefined);
          try {
            const response = await bookImport({ book_id: bookId, file });
            setResult(response.book_id);
            if (bookId) await revalidator.revalidate();
          } catch (error) {
            setError(
              error instanceof Error
                ? error.message
                : "The import failed. Please try again.",
            );
          } finally {
            setLoading(false);
            if (input.current) input.current.value = "";
          }
        }}
      />
      <button
        className="button button-primary"
        disabled={loading}
        onClick={() => input.current?.click()}
      >
        <Icon name="upload" size={17} />
        {loading ? "Importing…" : bookId ? "Upload a file" : "Import book"}
      </button>
      {error && (
        <span className="upload-message error-text" role="alert">
          {error}
        </span>
      )}
      {result && (
        <span className="upload-message" role="status">
          <Icon name="check" size={14} />
          Imported successfully
          {!bookId && <Link to={`/book/${result}`}>View book</Link>}
        </span>
      )}
    </div>
  );
}
