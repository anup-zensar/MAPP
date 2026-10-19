import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Tooltip } from 'radix-ui';
import type { ReactElement } from 'react';
import { createMemoryRouter, Outlet } from 'react-router';
import type { InitialEntry, RouteObject } from 'react-router';
import { RouterProvider } from 'react-router/dom';
import { expect } from 'vitest';
import { SessionProvider } from '@/app/SessionProvider';
import { SESSION_KEY } from '@/app/useSession';
import type { CurrentUser } from '@/app/useSession';
import { ToastProvider } from '@/components';

export interface RenderOptions {
  /**
   * The address the test starts on, with its query ("/plans?year=2026"), or a location with
   * router state ({ pathname: '/sign-in', state: { notice: 'signedOut' } }).
   */
  route?: InitialEntry;
  /** The signed-in person; null (the default) for a visitor who is not signed in. */
  user?: CurrentUser | null;
  /** The route path `ui` is shown at: every address ("*") unless given, for example "/sign-in". */
  path?: string;
}

/**
 * Renders a page or component the way the app does: with the session, server data (TanStack
 * Query), tooltips, messages and a router. The session is known from the start (`user`), so no
 * GET /api/me is needed. `ui` is shown at `path`; the other addresses show stand-in pages, so a
 * test can follow a redirect to the sign-in page ("Sign-in page") or to another page ("Another
 * page").
 *
 * Returns the router (router.state.location is the current address), the query client and a
 * user-event instance for clicking and typing.
 */
export function renderWithProviders(
  ui: ReactElement,
  { route = '/', user = null, path = '*' }: RenderOptions = {},
) {
  return renderRoutes(
    [
      {
        // Like the app's RootLayout: the session around every page.
        element: (
          <SessionProvider>
            <Outlet />
          </SessionProvider>
        ),
        children: [
          // React Router picks the most specific route, and the first of equally specific ones:
          // so the page under test wins over the stand-ins at its own path.
          { path, element: ui },
          { path: '/sign-in', element: <p>Sign-in page</p> },
          { path: '*', element: <p>Another page</p> },
        ],
      },
    ],
    { route, user },
  );
}

/**
 * Renders the given routes — usually the app's own, `routes` from '@/app/router' — with the same
 * providers as the app. Without `user`, the app asks GET /api/me as on a real page load, so the
 * test answers it with mockApi.
 */
export function renderRoutes(
  routes: RouteObject[],
  { route = '/', user }: Omit<RenderOptions, 'path'> = {},
) {
  const queryClient = new QueryClient({
    defaultOptions: {
      // A failed request shows its error at once, and nothing is cleared away during a test.
      queries: { retry: false, gcTime: Infinity },
    },
  });
  if (user !== undefined) {
    queryClient.setQueryData(SESSION_KEY, user);
  }

  const router = createMemoryRouter(routes, { initialEntries: [route] });

  render(
    <QueryClientProvider client={queryClient}>
      <Tooltip.Provider delayDuration={0}>
        <ToastProvider>
          <RouterProvider router={router} />
        </ToastProvider>
      </Tooltip.Provider>
    </QueryClientProvider>,
  );

  return { router, queryClient, user: userEvent.setup() };
}

/**
 * The messages area (toasts), once it shows a message that takes clicks. Radix lets the area take
 * clicks only after it has counted a new message, a moment after the text is on screen; so a test
 * that clicks a message's button (Try again, Undo) finds the area with this first.
 */
export async function findMessages(): Promise<HTMLElement> {
  const messages = screen.getByRole('region', { name: 'Messages (F8)' });
  await waitFor(() => expect(messages).not.toHaveStyle({ pointerEvents: 'none' }));
  return messages;
}
