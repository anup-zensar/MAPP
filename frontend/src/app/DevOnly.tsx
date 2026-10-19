import { useQuery } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { useMatches } from 'react-router';
import { api, call } from '@/api/client';
import { ErrorState, Skeleton } from '@/components';
import { NotFoundPage } from '@/features/placeholders/NotFoundPage';
import { usePageTitle } from '@/lib/usePageTitle';
import type { PageHandle } from './RequireSignIn';

/**
 * The gate in front of the developer pages (research R23, D-036): they exist only where demo
 * sign-in is on — the local and development environments. Anywhere else their address shows
 * "Page not found". The API's health answer says which it is; Vite's development flag is not
 * used, because the end-to-end tests run the production build.
 */
export function DevOnly({ children }: { children: ReactNode }) {
  const health = useQuery({
    queryKey: ['health'],
    queryFn: () => call(api.GET('/api/health')),
  });

  if (health.data) {
    return health.data.demoSignIn ? children : <NotFoundPage />;
  }
  return (
    <EnvironmentCheck
      error={health.isError ? health.error : null}
      onRetry={() => health.refetch()}
    />
  );
}

interface EnvironmentCheckProps {
  /** Why the API could not be asked; null while the question is on its way. */
  error: Error | null;
  onRetry: () => void;
}

/**
 * Shown until the API has said whether this is a demo environment: a loading placeholder, or the
 * error with Try again (FR-020). The tab already names the page that was asked for (FR-017).
 */
function EnvironmentCheck({ error, onRetry }: EnvironmentCheckProps) {
  const matches = useMatches();
  // Every developer route names its page in its handle, as every signed-in route does (router.tsx).
  usePageTitle((matches[matches.length - 1].handle as PageHandle).title);

  return error ? (
    <ErrorState error={error} onRetry={onRetry} />
  ) : (
    <Skeleton lines={4} height="18px" />
  );
}
