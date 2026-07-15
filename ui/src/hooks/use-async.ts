import { useEffect, useState } from "react";

export type AsyncResponse<T, E = Error> =
  | { loading: true; value: undefined; error: undefined }
  | { loading: false; value: T; error: undefined }
  | { loading: false; value: undefined; error: E };

export function useAsync<T, E = Error>(
  fn: (s: AbortSignal) => Promise<T>,
): AsyncResponse<T, E> {
  const [value, setValue] = useState<T>();
  const [error, setError] = useState<E>();

  useEffect(() => {
    const ac = new AbortController();
    setValue(undefined);
    setError(undefined);
    fn(ac.signal)
      .then((resp) => {
        if (ac.signal.aborted) {
          return;
        }
        setValue(resp);
        setError(undefined);
      })
      .catch((e) => {
        if (ac.signal.aborted) {
          return;
        }
        setValue(undefined);
        setError(e);
      });

    return () => {
      ac.abort();
    };
  }, [fn]);

  if (value !== undefined) {
    return { loading: false, value: value, error: undefined };
  }

  if (error !== undefined) {
    return { loading: false, value: undefined, error: error };
  }

  return { loading: true, value: undefined, error: undefined };
}
