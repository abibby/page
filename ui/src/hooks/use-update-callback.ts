import { useCallback, type InputEventHandler } from "react";

export function useUpdateCallback(fn: (s: string) => void): InputEventHandler<HTMLInputElement> {
  return useCallback((e) => {
    fn(e.currentTarget.value)
  }, [fn]);
}
