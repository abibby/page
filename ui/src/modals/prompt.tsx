import { ModalBody, type ModalProps } from "../components/modal";
import { Icon } from "../components/icon";

export type PromptProps = ModalProps<boolean> & { message: string };

export function Prompt({ close, message }: PromptProps) {
  return (
    <ModalBody close={() => close(false)}>
      <span className="empty-icon">
        <Icon name="book" size={28} />
      </span>
      <h2>A new addition to your shelf</h2>
      <p>
        Add <strong>{message}</strong> to your library? You can upload a file or
        find a download afterwards.
      </p>
      <div className="modal-actions">
        <button
          className="button button-secondary"
          onClick={() => close(false)}
          autoFocus
        >
          Cancel
        </button>
        <button className="button button-primary" onClick={() => close(true)}>
          <Icon name="plus" size={17} />
          Add to library
        </button>
      </div>
    </ModalBody>
  );
}
