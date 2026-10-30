import { useEffect } from 'react';
import type { ReactNode } from 'react';
import { Navigate, useLocation, useMatches } from 'react-router';
import { requestedPageTitle } from './page-handle';
import { SessionCheckPage } from './SessionCheckPage';
import { signInAddress, useSession } from './useSession';

/** Checks the session on every navigation; sends a signed-out visitor to sign in and back (FR-005). */
export function RequireSignIn({ children }: { children: ReactNode }) {
  const { status, refetch } = useSession();
  const location = useLocation();
  const matches = useMatches();

  useEffect(() => {
    refetch();
  }, [location.key, refetch]);

  if (status === 'loading' || status === 'unreachable') {
    return <SessionCheckPage title={requestedPageTitle(matches)} />;
  }
  if (status === 'signedOut') {
    return (
      <Navigate to={signInAddress(location.pathname + location.search + location.hash)} replace />
    );
  }
  return children;
}
