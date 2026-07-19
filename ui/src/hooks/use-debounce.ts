import { useEffect, useState } from "react";

export function useDebounce<T>(v: T, timeoutMs = 500): T {
  const [slow, setSlow] = useState(v);
  useEffect(() => {
    let unmounted = false;
    setTimeout(() => {
      if (unmounted) {
        return;
      }
      setSlow(v);
    }, timeoutMs);
    return () => {
      unmounted = true;
    };
  }, [v, timeoutMs]);
  return slow;
}
