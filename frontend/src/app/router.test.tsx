import { screen, within } from '@testing-library/react';
import type { ReactElement } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { mockApi } from '@/test/mockApi';
import { renderRoutes } from '@/test/render';
import { MAIN_ITEMS, MORE_ITEMS } from './navigation';
import { routes } from './router';
import type { CurrentUser } from './useSession';

const PRIYA: CurrentUser = {
  id: 'U-1002',
  name: 'Priya Raman',
  firstName: 'Priya',
  email: 'priya.raman@northstar.demo',
  jobTitle: 'Marketer',
  initials: 'PR',
  preferences: { theme: 'system', sidebarCollapsed: false },
};

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

  it.each([...MAIN_ITEMS, ...MORE_ITEMS].map((item) => [item.label, item.path]))(
    'opens %s from the sidebar address %s, highlighted, with its tab title (US2 scenario 1, SC-003)',
    async (label, path) => {
      mockApi({ 'GET /api/me': { body: PRIYA } });
      renderRoutes(routes, { route: path });

      const title = label === 'Home' ? 'Welcome back, Priya.' : label;
      expect(await screen.findByRole('heading', { level: 1, name: title })).toBeInTheDocument();
      const navigation = screen.getByRole('navigation', { name: 'Main' });
      expect(within(navigation).getByRole('link', { name: label })).toHaveAttribute(
        'aria-current',
        'page',
      );
      expect(document.title).toBe(`${label} · NorthStar`);
    },
  );

  it('opens My profile, which is not in the sidebar (FR-012)', async () => {
    mockApi({ 'GET /api/me': { body: PRIYA } });
    renderRoutes(routes, { route: '/profile' });

    expect(
      await screen.findByRole('heading', { level: 1, name: 'My profile' }),
    ).toBeInTheDocument();
    expect(document.title).toBe('My profile · NorthStar');
  });

  it.each([
    ['/', 'Home'],
    ['/plans', 'Plans'],
    ['/approvals', 'Approvals'],
    ['/all-work', 'All work'],
    ['/settings', 'Settings'],
    ['/recycle-bin', 'Recycle bin'],
    ['/activity-log', 'Activity log'],
    ['/profile', 'My profile'],
  ])(
    'names %s in the browser tab already while the session is checked (plan "Session")',
    async (path, title) => {
      mockApi({ 'GET /api/me': () => new Promise<never>(() => {}) });
      renderRoutes(routes, { route: path });

      await screen.findByRole('status');
      expect(document.title).toBe(`${title} · NorthStar`);
    },
  );
});
