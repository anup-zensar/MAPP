import { useEffect } from 'react';
import type { ReactNode } from 'react';
import { Navigate, useLocation, useMatches } from 'react-router';
import { requestedPageTitle } from './page-handle';
import { SessionCheckPage } from './SessionCheckPage';
import { signInAddress, useSession } from './useSession';

/**
 * The gate in front of every page that needs a signed-in person (FR-005, plan "Session"). It
 * checks the session again on every navigation, sends a visitor who is not signed in to the
 * sign-in page — and back here afterwards — and shows `children` (the app layout) once signed in.
 */
export function RequireSignIn({ children }: { children: ReactNode }) {
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
  return children;
}
