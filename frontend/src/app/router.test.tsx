import { screen, within } from '@testing-library/react';
import type { ReactElement } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { HOME_ANSWERS } from '@/testing/fixtures/home';
import { mockApi } from '@/testing/mockApi';
import { renderRoutes } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { COMING_SOON_SECTIONS, MAIN_ITEMS, MORE_ITEMS } from './navigation';
import { routes } from './router';

const PRIYA = testUser();

const ENVIRONMENT_LOCAL = { environment: 'local', demoSignIn: true };

function BrokenPage(): never {
  throw new Error('A programming mistake');
}

/** The app's root route — its layout and its error page — with one test page inside. */
function renderInRoot(page: ReactElement) {
  const [root] = routes;
  renderRoutes(
    [
      {
        path: '/',
        element: root.element,
        errorElement: root.errorElement,
        children: [{ index: true, element: page }],
      },
    ],
    { user: null },
  );
}

afterEach(() => {
  vi.restoreAllMocks();
  localStorage.clear();
  delete document.documentElement.dataset.theme;
});

describe('routes', () => {
  it('shows the pages inside the root layout (plan "Router")', () => {
    renderInRoot(<h1>A page</h1>);

    expect(screen.getByRole('heading', { name: 'A page' })).toBeInTheDocument();
  });

  it('catches an error in any page and shows the error page, never a blank screen (FR-020)', () => {
    // React reports the thrown error on the console; the test only looks at the screen.
    vi.spyOn(console, 'error').mockImplementation(() => {});

    renderInRoot(<BrokenPage />);

    expect(screen.getByRole('alert')).toHaveTextContent('Something went wrong. Please try again.');
    expect(screen.getByRole('button', { name: 'Try again' })).toBeInTheDocument();
  });

  // Home and the sections still to come; each section of features 002–008 is checked in the
  // routes.test.tsx of its feature folder, which its feature keeps up to date.
  it.each(
    [...MAIN_ITEMS, ...MORE_ITEMS]
      .filter(
        (item) =>
          item.path === '/' || COMING_SOON_SECTIONS.some((section) => section.path === item.path),
      )
      .map((item) => [item.label, item.path]),
  )(
    'opens %s from the sidebar address %s, highlighted, with its tab title (US2 scenario 1, SC-003)',
    async (label, path) => {
      mockApi({ 'GET /api/v1/me': { body: PRIYA }, ...HOME_ANSWERS });
      renderRoutes(routes, { route: path });

      const title = label === 'Home' ? 'Welcome back, Priya.' : label;
      expect(await screen.findByRole('heading', { level: 1, name: title })).toBeInTheDocument();
      const navigation = screen.getByRole('navigation', { name: 'Main' });
      expect(within(navigation).getByRole('link', { name: label })).toHaveAttribute(
        'aria-current',
        'page',
      );
      expect(document.title).toBe(`${label} · MAPP`);
    },
  );

  it.each([
    ['/', 'Home'],
    ['/plans', 'Plans'],
    ['/approvals', 'Approvals'],
    ['/all-work', 'All work'],
    ['/budgets', 'Global budgets'],
    ['/budgets/GB-1001', 'Global budget'],
    ['/financials', 'Financials'],
    ['/financials/transactions/FT-1003', 'Financials'],
    ['/settings', 'Settings'],
    ['/recycle-bin', 'Recycle bin'],
    ['/activity-log', 'Activity log'],
    ['/profile', 'My profile'],
    ['/plans/P-1001/campaigns/C-1001', 'Plans'],
    ['/activities/A-1001/line-items/new', 'New line item'],
    ['/dev/sample-records', 'Sample records'],
    ['/dev/components', 'Shared building blocks'],
  ])(
    'names %s in the browser tab already while the session is checked (plan "Session")',
    async (path, title) => {
      mockApi({ 'GET /api/v1/me': () => new Promise<never>(() => {}) });
      renderRoutes(routes, { route: path });

      await screen.findByRole('status');
      expect(document.title).toBe(`${title} · MAPP`);
    },
  );

  it.each([
    ['/dev/sample-records', 'Sample records'],
    ['/dev/components', 'Shared building blocks'],
  ])(
    'opens the developer page %s in the app layout, outside the navigation, where demo sign-in is on (research R23)',
    async (path, title) => {
      mockApi({
        'GET /api/v1/me': { body: PRIYA },
        'GET /api/v1/environment': { body: ENVIRONMENT_LOCAL },
        'GET /api/v1/sample-folders': { body: { items: [], total: 0 } },
        'GET /api/v1/recycle-bin': { body: { items: [], total: 0 } },
        'GET /api/v1/activity-log': { body: { items: [], total: 0 } },
      });
      renderRoutes(routes, { route: path });

      expect(await screen.findByRole('heading', { level: 1, name: title })).toBeInTheDocument();
      const navigation = screen.getByRole('navigation', { name: 'Main' });
      expect(within(navigation).queryByRole('link', { name: title })).not.toBeInTheDocument();
      expect(document.title).toBe(`${title} · MAPP`);
    },
  );

  it.each(['/dev/sample-records', '/dev/components'])(
    'shows "Page not found" at %s where demo sign-in is off (FR-009, D-036)',
    async (path) => {
      const api = mockApi({
        'GET /api/v1/me': { body: PRIYA },
        'GET /api/v1/environment': {
          body: { ...ENVIRONMENT_LOCAL, environment: 'production', demoSignIn: false },
        },
      });
      renderRoutes(routes, { route: path });

      expect(
        await screen.findByRole('heading', { level: 1, name: 'Page not found' }),
      ).toBeInTheDocument();
      expect(screen.getByRole('navigation', { name: 'Main' })).toBeInTheDocument();
      // The developer route asked the API which environment this is.
      expect(api.requests.map((request) => request.path)).toContain('/api/v1/environment');
    },
  );
});
