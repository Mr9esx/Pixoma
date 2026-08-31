import { useCallback, useSyncExternalStore } from "react";

export function useMedia(query: string): boolean {
  const subscribe = useCallback(
    (onStoreChange: () => void) => {
      const matchMedia = window.matchMedia(query);
      matchMedia.addEventListener("change", onStoreChange);

      return () => {
        matchMedia.removeEventListener("change", onStoreChange);
      };
    },
    [query],
  );

  const getSnapshot = useCallback(
    () => window.matchMedia(query).matches,
    [query],
  );

  return useSyncExternalStore(
    subscribe,
    getSnapshot,
    useCallback(() => true, []),
  );
}
