import { NotFoundPage } from '@/features/placeholders/NotFoundPage';
import { AppLayout } from './AppLayout';
import { PublicLayout } from './PublicLayout';
import { SessionCheckPage } from './SessionCheckPage';
import { useSession } from './useSession';

/**
 * Any address MAPP does not know (FR-015, plan "Router"): "Page not found" inside the app
 * layout for a signed-in person, in the public layout for everyone else — never a redirect to
 * the sign-in page.
 */
export function NotFoundRoute() {
  const { status } = useSession();

  if (status === 'loading' || status === 'unreachable') {
    return <SessionCheckPage title="Page not found" />;
  }
  if (status === 'signedIn') {
    return (
      <AppLayout>
        <NotFoundPage />
      </AppLayout>
    );
  }
  return (
    <PublicLayout>
      <NotFoundPage />
    </PublicLayout>
  );
}
