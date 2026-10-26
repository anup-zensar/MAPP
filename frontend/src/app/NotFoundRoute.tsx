import { NotFoundPage } from '@/features/placeholders';
import { PublicLayout } from '@/ui/PublicLayout/PublicLayout';
import { SessionCheckPage } from '@/session/SessionCheckPage';
import { useSession } from '@/session/useSession';
import { AppLayout } from './AppLayout';

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
