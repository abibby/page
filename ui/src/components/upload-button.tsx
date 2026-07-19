import { useCallback, useState } from "react";
import { bookImport } from "../api/api";
import { Link } from "react-router";
import { routePath } from "../routes";

export type UploadButtonProps = {
  bookId?: number;
};

export function UploadButton(props: UploadButtonProps) {
  const [loading, setLoading] = useState(false);
  const [newBookID, setNewBookID] = useState<number>();

  const uploadFiles = useCallback(async () => {
    const files = await openFilePicker();
    if (!files) {
      return;
    }
    setLoading(true);
    try {
      const resp = await bookImport({
        book_id: props.bookId,
        file: files[0],
      });
      setNewBookID(resp.book_id);
    } finally {
      setLoading(false);
    }
  }, [props.bookId]);
  return (
    <div>
      <button onClick={uploadFiles} disabled={loading}>
        Upload Book
      </button>
      {newBookID && (
        <Link to={routePath("book.view", { bookId: newBookID })}>
          Open new book
        </Link>
      )}
    </div>
  );
}

type OpenFilePickerOptions = {
  multiple?: boolean;
};

function openFilePicker(
  options: OpenFilePickerOptions = {},
): Promise<FileList | null> {
  return new Promise((resolve) => {
    const input = document.createElement("input");
    input.type = "file";
    input.style.display = "none";

    input.multiple = options.multiple ?? false;

    input.onchange = () => {
      resolve(input.files);
      input.remove();
    };

    document.body.appendChild(input);

    if ("showPicker" in HTMLInputElement.prototype) {
      input.showPicker();
    } else {
      input.click();
    }
  });
}
