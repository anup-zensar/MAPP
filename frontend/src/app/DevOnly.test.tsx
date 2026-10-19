import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/test/mockApi';
import { renderRoutes } from '@/test/render';
import { DevOnly } from './DevOnly';
import type { PageHandle } from './RequireSignIn';

const HEALTH_LOCAL = { status: 'ok', environment: 'local', demoSignIn: true };
const HEALTH_PRODUCTION = { status: 'ok', environment: 'production', demoSignIn: false };

const UNREACHABLE =
  "NorthStar can't reach its server right now. Check that it is running and try again.";

/** A developer page behind DevOnly, named in its route's handle as in router.tsx. */
function renderDeveloperPage() {
  return renderRoutes(
    [
      {
        path: '/dev/example',
        element: (
          <DevOnly>
            <h1>Example developer page</h1>
          </DevOnly>
        ),
        handle: { title: 'Example' } satisfies PageHandle,
      },
    ],
    { route: '/dev/example' },
  );
}

describe('DevOnly', () => {
  it('shows the developer page where demo sign-in is on: local and development (research R23)', async () => {
    mockApi({ 'GET /api/health': { body: HEALTH_LOCAL } });
    renderDeveloperPage();

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Example developer page' }),
    ).toBeInTheDocument();
  });

  it('shows "Page not found" where demo sign-in is off (FR-009, D-036)', async () => {
    mockApi({ 'GET /api/health': { body: HEALTH_PRODUCTION } });
    renderDeveloperPage();

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Page not found' }),
    ).toBeInTheDocument();
    expect(screen.queryByText('Example developer page')).not.toBeInTheDocument();
    expect(document.title).toBe('Page not found · NorthStar');
  });

  it('shows a loading placeholder, with the page’s name in the tab, while it asks the API (FR-017)', async () => {
    mockApi({ 'GET /api/health': () => new Promise<never>(() => {}) });
    renderDeveloperPage();

    expect(await screen.findByRole('status')).toHaveTextContent('Loading…');
    expect(document.title).toBe('Example · NorthStar');
  });

  it('says when the API cannot be reached, and shows the page after Try again (FR-020)', async () => {
    let checks = 0;
    mockApi({
      'GET /api/health': () =>
        ++checks === 1 ? Promise.reject(new TypeError('Failed to fetch')) : { body: HEALTH_LOCAL },
    });
    const { user } = renderDeveloperPage();

    expect(await screen.findByRole('alert')).toHaveTextContent(UNREACHABLE);
    expect(document.title).toBe('Example · NorthStar');
    await user.click(screen.getByRole('button', { name: 'Try again' }));

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Example developer page' }),
    ).toBeInTheDocument();
  });
});
