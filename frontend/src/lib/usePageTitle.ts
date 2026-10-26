import { useEffect } from 'react';

/**
 * Names the page in the browser tab: usePageTitle('Plans') shows "Plans · MAPP" (FR-017).
 * null leaves the tab alone — for a page while a side panel with its own title is open over it
 * (feature 005's panels over the plan page).
 */
export function usePageTitle(page: string | null): void {
  useEffect(() => {
    if (page !== null) {
      document.title = `${page} · MAPP`;
    }
  }, [page]);
}
