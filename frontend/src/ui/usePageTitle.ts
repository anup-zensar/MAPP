import { useEffect } from 'react';

/** null leaves the tab alone, for a page under a side panel that names the tab itself (feature 005). */
export function usePageTitle(page: string | null): void {
  useEffect(() => {
    if (page !== null) {
      document.title = `${page} · SP&A One`;
    }
  }, [page]);
}
