import {
  useState,
  useEffect,
  useCallback,
  type Dispatch,
  type SetStateAction,
} from "react";

/**
 * Syncs string state with a URL query parameter.
 *
 * @param key - The query parameter name (e.g., 'search')
 * @param initialValue - Fallback value if the parameter isn't in the URL
 * @returns Standard [value, setValue] tuple
 */
export function useQueryString(
  key: string,
  initialValue: string = "",
): [string, Dispatch<SetStateAction<string>>] {
  // 1. Initialize state directly from the URL if available
  const [value, setValue] = useState<string>(() => {
    if (typeof window === "undefined") return initialValue;

    const searchParams = new URLSearchParams(window.location.search);
    return searchParams.get(key) ?? initialValue;
  });

  // 2. Wrap the state setter to also update the URL
  const setQueryString = useCallback<Dispatch<SetStateAction<string>>>(
    (newValue) => {
      // Support functional updates (e.g., prev => prev + 'a')
      const resolvedValue =
        typeof newValue === "function" ? newValue(value) : newValue;

      setValue(resolvedValue);

      if (typeof window !== "undefined") {
        const url = new URL(window.location.href);

        // Clean up the URL if the value is empty
        if (
          resolvedValue === "" ||
          resolvedValue === null ||
          resolvedValue === undefined
        ) {
          url.searchParams.delete(key);
        } else {
          url.searchParams.set(key, resolvedValue);
        }

        console.log(url.toString());

        // Use replaceState to avoid cluttering the browser history stack
        window.history.replaceState({}, "", url.toString());
      }
    },
    [key, value],
  );

  // 3. Listen for browser back/forward buttons to keep state in sync
  useEffect(() => {
    const handlePopState = () => {
      const searchParams = new URLSearchParams(window.location.search);
      setValue(searchParams.get(key) ?? initialValue);
    };

    window.addEventListener("popstate", handlePopState);
    return () => window.removeEventListener("popstate", handlePopState);
  }, [key, initialValue]);

  return [value, setQueryString];
}
