import { useCallback, useEffect, useRef, useState } from 'react';

// Runs `fetcher` (a function returning a promise of the API response) and tracks
// loading / error state. Re-runs whenever a value in `deps` changes or reload() is called.
export function useFetch(fetcher, deps = []) {
  const [state, setState] = useState({ data: null, meta: null, loading: true, error: '' });
  const [tick, setTick] = useState(0);
  const fetcherRef = useRef(fetcher);
  fetcherRef.current = fetcher;

  useEffect(() => {
    let cancelled = false;
    setState((s) => ({ ...s, loading: true, error: '' }));
    fetcherRef
      .current()
      .then((res) => {
        if (!cancelled) setState({ data: res.data, meta: res.meta || null, loading: false, error: '' });
      })
      .catch((err) => {
        if (!cancelled) setState({ data: null, meta: null, loading: false, error: err.message });
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...deps, tick]);

  const reload = useCallback(() => setTick((t) => t + 1), []);
  return { ...state, reload };
}
