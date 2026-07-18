import { useCallback } from "react";
import { ModalBody, type ModalProps } from "../components/modal";

export type PromptProps = ModalProps<boolean> & {
    message: string
};

export function Prompt(props: PromptProps) {
  const accept = useCallback(() => {
    props.close(true);
  }, [props.close]);
  const reject = useCallback(() => {
    props.close(false);
  }, [props.close]);
  return (
    <ModalBody close={reject}>
      <p>
        {props.message}
      </p>
      <button onClick={accept}>Accept</button>
      <button onClick={reject}>Reject</button>
    </ModalBody>
  );
}
