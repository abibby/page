// oxlint-disable react/only-export-components
import React, {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  useEffect,
  useRef,
  type PropsWithChildren,
} from "react";

export type ModalProps<T> = {
  close(result: T): void;
};

type ModalData = {
  id: number;
  Component: React.ComponentType<ModalProps<unknown>>;
  props: Record<string, unknown>;
  resolve(result: any): void;
};

type ModalContext = {
  openModal(data: ModalData): void;
  nextID(): number;
};

const Context = createContext<ModalContext | undefined>(undefined);

export function ModalProvider(props: PropsWithChildren) {
  const [modals, setModals] = useState<ModalData[]>([]);
  const data = useMemo<ModalContext>(() => {
    let id = 0;
    return {
      openModal(data) {
        setModals((m) => m.concat([data]));
      },
      nextID() {
        id++;
        return id;
      },
    };
  }, []);

  const modalElements = useMemo(() => {
    return modals.map(({ Component, ...modal }) => {
      const closeModal = (result: unknown) => {
        setModals((ms) => ms.filter((m) => m.id != modal.id));
        modal.resolve(result);
      };
      return <Component key={modal.id} close={closeModal} {...modal.props} />;
    });
  }, [modals]);

  return (
    <Context.Provider value={data}>
      {props.children}
      {modalElements}
    </Context.Provider>
  );
}

export function useModal() {
  const data = useContext(Context);
  if (data === undefined) {
    throw new Error("you can only use useModal inside a ModalProvider");
  }
  return {
    async openModal<TProps extends ModalProps<TReturn>, TReturn>(
      Component: React.ComponentType<TProps>,
      props: Omit<TProps, keyof ModalProps<TReturn>>,
    ): Promise<TReturn> {
      return new Promise((resolve) => {
        data.openModal({
          id: data.nextID(),
          Component: Component as React.ComponentType<ModalProps<unknown>>,
          props: props,
          resolve: (result) => resolve(result),
        });
      });
    },
  };
}

export type ModalBodyProps = PropsWithChildren<{
  close?: (result: undefined) => void;
}>;

export function ModalBody(props: ModalBodyProps) {
  const dialog = useRef<HTMLDialogElement>(null);
  const close = useCallback(() => props.close?.(undefined), [props]);
  useEffect(() => {
    const element = dialog.current;
    element?.showModal();
    return () => element?.close();
  }, []);
  return (
    <dialog
      ref={dialog}
      className="modal"
      aria-label="Add book to library"
      onCancel={(event) => {
        event.preventDefault();
        close();
      }}
      onClick={(event) => {
        if (event.target === event.currentTarget) close();
      }}
    >
      <div className="modal-content">{props.children}</div>
    </dialog>
  );
}
