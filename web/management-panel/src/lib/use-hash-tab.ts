import * as React from "react";

/**
 * Deep-linkable tab state stored in the hash query string (e.g.
 * `#/routing?tab=upstreams`). The hash router (`app/router.tsx`) ignores the
 * query portion when resolving routes, so writing `?tab=` here keeps the active
 * route stable while making sub-tabs survive reloads and back/forward.
 */
export function useHashTab<const T extends readonly string[]>(
  tabs: T,
  fallback: T[number],
): [T[number], (next: T[number]) => void] {
  const read = React.useCallback((): T[number] => {
    const hash = window.location.hash.replace(/^#/, "");
    const queryIndex = hash.indexOf("?");
    if (queryIndex === -1) return fallback;
    const params = new URLSearchParams(hash.slice(queryIndex + 1));
    const value = params.get("tab");
    return value && (tabs as readonly string[]).includes(value)
      ? (value as T[number])
      : fallback;
  }, [tabs, fallback]);

  const [tab, setTab] = React.useState<T[number]>(read);

  // Keep state in sync when the hash changes (back/forward, route switch).
  React.useEffect(() => {
    const onHashChange = () => setTab(read());
    window.addEventListener("hashchange", onHashChange);
    return () => window.removeEventListener("hashchange", onHashChange);
  }, [read]);

  const navigate = React.useCallback((next: T[number]) => {
    const hash = window.location.hash.replace(/^#/, "");
    const queryIndex = hash.indexOf("?");
    const path = queryIndex === -1 ? hash : hash.slice(0, queryIndex);
    const params = new URLSearchParams(queryIndex === -1 ? "" : hash.slice(queryIndex + 1));
    params.set("tab", next);
    window.location.hash = `${path}?${params.toString()}`;
  }, []);

  return [tab, navigate];
}
