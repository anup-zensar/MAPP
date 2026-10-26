import { useSession } from '@/session/useSession';
import type { CurrentUser } from '@/session/useSession';
import { SessionProvider } from '@/session/SessionProvider';
import { renderRoutes } from '../render';
import { apiError, reply } from '../replies';
import { NO_DELEGATIONS } from './approvals';
import { testUser } from './users';

export const PRIYA: CurrentUser = testUser();

export const DANIEL: CurrentUser = testUser({
  id: 'U-1003',
  name: 'Daniel Okoro',
  firstName: 'Daniel',
  email: 'daniel.okoro@shell.com',
  jobTitle: 'Marketing team leader',
  initials: 'DO',
});

export const SIGN_IN_REQUIRED = apiError('sign_in_required', 'Please sign in to continue.');
export const SESSION_EXPIRED = apiError('session_expired', 'Please sign in again');

/** The Plans page's request, answered with no plans: these tests use /plans as a signed-in page. */
export const NO_PLANS = [
  reply('GET /api/v1/plans', {
    body: {
      items: [],
      total: 0,
      summary: {
        plans: 0,
        draft: 0,
        published: 0,
        plannedCost: { amount: '0.00', currency: 'USD' },
        budget: { amount: '0.00', currency: 'USD' },
      },
      years: [],
    },
  }),
];

/** The Approvals page's request, answered with no rows. */
export const NO_APPROVALS = [
  reply('GET /api/v1/approvals', { body: { items: [], total: 0 } }),
  reply('GET /api/v1/approval-delegations', NO_DELEGATIONS),
];

/** Shows what useSession() says, with a button that checks the session again. */
export function SessionProbe() {
  const session = useSession();
  return (
    <>
      <p>Status: {session.status}</p>
      <p>Person: {session.user?.name ?? 'nobody'}</p>
      <button type="button" onClick={session.refetch}>
        Check again
      </button>
    </>
  );
}

/** The probe under the session, on a fresh page load: nothing is known until GET /api/v1/me answers. */
export function renderProbe() {
  return renderRoutes([
    {
      path: '*',
      element: (
        <SessionProvider>
          <SessionProbe />
        </SessionProvider>
      ),
    },
  ]);
}

/** A promise the test resolves when it wants, to keep several requests waiting at once. */
export function gate() {
  let open = () => {};
  const opened = new Promise<void>((resolve) => {
    open = resolve;
  });
  return { opened, open };
}
