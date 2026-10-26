import { useEffect, useRef, useState } from 'react';

/** The value once it has stayed the same (by its JSON) for `delay` ms. */
export function useDebounced<T>(value: T, delay: number): T {
  const [settled, setSettled] = useState(value);
  const latest = useRef(value);
  const key = JSON.stringify(value);

  useEffect(() => {
    latest.current = value;
  });

  useEffect(() => {
    const timer = window.setTimeout(() => setSettled(latest.current), delay);
    return () => window.clearTimeout(timer);
  }, [key, delay]);

  return settled;
}
