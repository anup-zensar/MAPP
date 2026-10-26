import { useRouteError } from 'react-router';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { PublicLayout } from '@/ui/PublicLayout/PublicLayout';

/**
 * Shown instead of a page that failed to draw, so an unexpected error never leaves a blank screen
 * (plan "Router"). A programming mistake shows the general message "Something went wrong. Please
 * try again."; Try again reloads the page.
 *
 * It replaces the whole app, the session included, so it leaves out the theme switch, which
 * needs the session.
 */
export function RouteErrorPage() {
  const error = useRouteError();

  return (
    <PublicLayout withThemeSwitch={false}>
      <ErrorState error={error} onRetry={() => window.location.reload()} />
    </PublicLayout>
  );
}
