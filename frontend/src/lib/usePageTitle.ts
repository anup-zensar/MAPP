import { useEffect } from 'react';

/** Names the page in the browser tab: usePageTitle('Plans') shows "Plans · NorthStar" (FR-017). */
export function usePageTitle(page: string): void {
  useEffect(() => {
    document.title = `${page} · NorthStar`;
  }, [page]);
}
