import type { UIMatch } from 'react-router';

/** What every route inside RequireSignIn says about its page, in its `handle` (router.tsx). */
export interface PageHandle {
  /** The page's name in the browser tab, shown already while the session is being checked. */
  title: string;
}

/** The tab title of the page asked for: the `handle` of the deepest matching route. */
export function requestedPageTitle(matches: UIMatch[]): string {
  // router.test.tsx checks that every route inside RequireSignIn has one.
  const handle = matches.at(-1)?.handle;
  return isPageHandle(handle) ? handle.title : '';
}

function isPageHandle(handle: unknown): handle is PageHandle {
  return (
    typeof handle === 'object' &&
    handle !== null &&
    'title' in handle &&
    typeof handle.title === 'string'
  );
}
