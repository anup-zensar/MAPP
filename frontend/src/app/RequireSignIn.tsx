import { useEffect } from 'react';
import { Navigate, Outlet, useLocation, useMatches } from 'react-router';
import type { UIMatch } from 'react-router';
import { AppLayout } from './AppLayout';
import { SessionCheckPage } from './SessionCheckPage';
import { signInAddress, useSession } from './useSession';

/** What every route inside RequireSignIn says about its page, in its `handle` (router.tsx). */
export interface PageHandle {
  /** The page's name in the browser tab, shown already while the session is being checked. */
  title: string;
}

/**
 * The gate in front of every page that needs a signed-in person (FR-005, plan "Session"). It
 * checks the session again on every navigation, sends a visitor who is not signed in to the
 * sign-in page — and back here afterwards — and shows the page inside the app layout.
 */
export function RequireSignIn() {
  const { status, refetch } = useSession();
  const location = useLocation();
  const matches = useMatches();

  // Every navigation asks the API again, so a session that ended elsewhere is noticed at once.
  useEffect(() => {
    refetch();
  }, [location.key, refetch]);

  if (status === 'loading' || status === 'unreachable') {
    return <SessionCheckPage title={requestedPageTitle(matches)} />;
  }
  if (status === 'signedOut') {
    // Replace: Back must not return to a page that only sends people here again.
    return <Navigate to={signInAddress(location.pathname + location.search)} replace />;
  }
  return (
    <AppLayout>
      <Outlet />
    </AppLayout>
  );
}

/** The tab title of the page asked for: the `handle` of the deepest matching route. */
function requestedPageTitle(matches: UIMatch[]): string {
  // router.test.tsx checks that every route inside RequireSignIn has one.
  return (matches[matches.length - 1].handle as PageHandle).title;
}
