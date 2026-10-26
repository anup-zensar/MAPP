import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/testing/mock-api';
import { renderRoutes } from '@/testing/render';
import type { PageHandle } from '@/session/page-handle';
import { reply } from '@/testing/replies';
import { ENVIRONMENT_LOCAL, ENVIRONMENT_PRODUCTION } from '@/testing/fixtures/environment';
import { DevOnly } from './DevOnly';

const UNREACHABLE =
  "MAPP can't reach its server right now. Check that it is running and try again.";

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
    mockApi([reply('GET /api/v1/environment', { body: ENVIRONMENT_LOCAL })]);
    renderDeveloperPage();

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Example developer page' }),
    ).toBeInTheDocument();
  });

  it('shows "Page not found" where demo sign-in is off (FR-009, D-036)', async () => {
    mockApi([reply('GET /api/v1/environment', { body: ENVIRONMENT_PRODUCTION })]);
    renderDeveloperPage();

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Page not found' }),
    ).toBeInTheDocument();
    expect(screen.queryByText('Example developer page')).not.toBeInTheDocument();
    expect(document.title).toBe('Page not found · MAPP');
  });

  it('shows a loading placeholder, with the page’s name in the tab, while it asks the API (FR-017)', async () => {
    mockApi([reply('GET /api/v1/environment', () => new Promise<never>(() => {}))]);
    renderDeveloperPage();

    expect(await screen.findByRole('status')).toHaveTextContent('Loading…');
    expect(document.title).toBe('Example · MAPP');
  });

  it('says when the API cannot be reached, and shows the page after Try again (FR-020)', async () => {
    let checks = 0;
    mockApi([
      reply('GET /api/v1/environment', () =>
        ++checks === 1
          ? Promise.reject(new TypeError('Failed to fetch'))
          : { body: ENVIRONMENT_LOCAL },
      ),
    ]);
    const { user } = renderDeveloperPage();

    expect(await screen.findByRole('alert')).toHaveTextContent(UNREACHABLE);
    expect(document.title).toBe('Example · MAPP');
    await user.click(screen.getByRole('button', { name: 'Try again' }));

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Example developer page' }),
    ).toBeInTheDocument();
  });
});
