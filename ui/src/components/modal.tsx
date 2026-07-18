import React, {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type PropsWithChildren,
} from "react";
import styles from './modal.module.css'

export type ModalProps<T> = {
  close(result: T): void;
};

type ModalData = {
  id: number;
  Component: React.ComponentType<ModalProps<unknown>>;
  props: Record<string, unknown>;
  resolve(result: any): void
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
        modal.resolve(result)
      };
      return <Component key={modal.id} close={closeModal} {...modal.props}/>;
    });
  }, [modals, data]);

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
            resolve: result => resolve(result),
        });
      });
    },
  };
}

export type ModalBodyProps = PropsWithChildren<{
    close?: (result: undefined) => void
}>

export function ModalBody(props: ModalBodyProps) {
    const close = useCallback(() => {
        props.close?.(undefined)
    }, [props.close])

    return <div>
        <div className={styles.screen} onClick={close}></div>
        <div className={styles.modal}>
            {props.children}
        </div>
    </div>
}

