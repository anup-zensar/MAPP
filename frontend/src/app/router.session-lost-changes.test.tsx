import { screen, waitFor } from '@testing-library/react';
import { useState } from 'react';
import type { RouteObject } from 'react-router';
import { afterEach, describe, expect, it } from 'vitest';
import { SignInPage } from '@/features/sign-in';
import { patchPreferences } from '@/session/api';
import { RequireSignIn } from '@/session/RequireSignIn';
import { PRIYA, SESSION_EXPIRED } from '@/testing/fixtures/session';
import { DEMO_PEOPLE } from '@/testing/fixtures/sign-in';
import { mockApi } from '@/testing/mock-api';
import { renderRoutes } from '@/testing/render';
import { reply } from '@/testing/replies';
import { usePageTitle } from '@/ui/usePageTitle';
import { useUnsavedChanges } from '@/ui/useUnsavedChanges';
import { RootLayout } from './RootLayout';

afterEach(() => {
  localStorage.clear();
  document.title = '';
});

/** A form with a name field; Save asks the API, which says the session has ended. */
function Form({ title }: { title?: string }) {
  usePageTitle('New plan');
  const [name, setName] = useState('');
  const { dialog } = useUnsavedChanges(name !== '', title);
  return (
    <>
      <label htmlFor="name">Name</label>
      <input id="name" value={name} onChange={(event) => setName(event.target.value)} />
      <button type="button" onClick={() => patchPreferences({ theme: 'dark' }).catch(() => {})}>
        Save
      </button>
      {dialog}
    </>
  );
}

function routesWith(form: RouteObject['element']): RouteObject[] {
  return [
    {
      element: <RootLayout />,
      children: [
        { path: '/sign-in', element: <SignInPage /> },
        { path: '/form', element: <RequireSignIn>{form}</RequireSignIn> },
      ],
    },
  ];
}

function lostSessionApi() {
  mockApi([
    reply('GET /api/v1/me', { body: PRIYA }),
    reply('PATCH /api/v1/me/preferences', { status: 401, body: SESSION_EXPIRED }),
    reply('GET /api/v1/auth/demo-people', { body: DEMO_PEOPLE }),
  ]);
}

describe('A session lost with unsaved changes (038 FR-031, D-357)', () => {
  it('038 FR-031: names the form whose changes were not saved and keeps the #fragment of the way back', async () => {
    lostSessionApi();
    const { user, router } = renderRoutes(routesWith(<Form title="New team" />), {
      route: '/form?tab=1#basics',
    });

    await user.type(await screen.findByLabelText('Name'), 'Ops');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/sign-in'));
    expect(
      await screen.findByText(
        "Your session ended, so your changes in 'New team' weren't saved. Please sign in again",
      ),
    ).toBeInTheDocument();
    expect(router.state.location.search).toBe('?next=%2Fform%3Ftab%3D1%23basics');
  });

  it('038 FR-031: a full-page form is named by its page title', async () => {
    lostSessionApi();
    const { user } = renderRoutes(routesWith(<Form />), { route: '/form' });

    await user.type(await screen.findByLabelText('Name'), 'Ops');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(
      await screen.findByText(
        "Your session ended, so your changes in 'New plan' weren't saved. Please sign in again",
      ),
    ).toBeInTheDocument();
  });

  it('038 FR-031: without typed changes the notice stays "Please sign in again"', async () => {
    lostSessionApi();
    const { user } = renderRoutes(routesWith(<Form />), { route: '/form' });

    await user.click(await screen.findByRole('button', { name: 'Save' }));

    expect(await screen.findByText('Please sign in again')).toBeInTheDocument();
  });
});
