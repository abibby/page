import { useCallback } from "react";
import { ModalBody, type ModalProps } from "../components/modal";

export type PromptProps = ModalProps<boolean> & {
  message: string;
};

export function Prompt({ close, message }: PromptProps) {
  const accept = useCallback(() => {
    close(true);
  }, [close]);
  const reject = useCallback(() => {
    close(false);
  }, [close]);
  return (
    <ModalBody close={reject}>
      <p>{message}</p>
      <button onClick={accept}>Accept</button>
      <button onClick={reject}>Reject</button>
    </ModalBody>
  );
}
