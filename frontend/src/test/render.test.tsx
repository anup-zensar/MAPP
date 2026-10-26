import { act, screen, within } from '@testing-library/react';
import { useState } from 'react';
import { Link, useLocation } from 'react-router';
import { afterEach, describe, expect, it } from 'vitest';
import { SessionProvider } from '@/app/SessionProvider';
import { useSession } from '@/app/useSession';
import type { CurrentUser } from '@/app/useSession';
import { Button } from '@/ui/Button/Button';
import { useToast } from '@/ui/Toast/useToast';
import { Tooltip } from '@/ui/Tooltip/Tooltip';
import { mockApi } from './mockApi';
import { findMessages, renderRoutes, renderWithProviders } from './render';
import { testUser } from '@/test/users';

const PRIYA: CurrentUser = testUser();

/** Uses every provider: the router, tooltips and messages. */
function Page() {
  const location = useLocation();
  const { show } = useToast();
  return (
    <>
      <p>
        At {location.pathname}
        {location.search}
      </p>
      <Tooltip content="Shows a message">
        <Button onClick={() => show({ message: 'Hello' })}>Say hello</Button>
      </Tooltip>
      <Link to="/sign-in?next=%2Fplans">Sign in again</Link>
    </>
  );
}

/** Shows who the session says is signed in. */
function SessionName() {
  const { user } = useSession();
  return <p>Signed in: {user?.name ?? 'nobody'}</p>;
}

/** Shows the router state of the current location. */
function RouterState() {
  const location = useLocation();
  return <p>State: {JSON.stringify(location.state)}</p>;
}

/** Shows a message with Try again a moment after Save, as after an API call that failed. */
function SaveLater() {
  const { show } = useToast();
  const [triedAgain, setTriedAgain] = useState(false);

  function save() {
    window.setTimeout(() => {
      show({
        message: 'Not saved',
        action: { label: 'Try again', altText: 'Try again', onClick: () => setTriedAgain(true) },
      });
    }, 0);
  }

  return (
    <>
      <Button onClick={save}>Save</Button>
      {triedAgain && <p>Tried again</p>}
    </>
  );
}

afterEach(() => {
  localStorage.clear();
  delete document.documentElement.dataset.theme;
});

describe('renderWithProviders', () => {
  it('shows the page at the address the test starts on', () => {
    renderWithProviders(<Page />, { route: '/plans?year=2026' });

    expect(screen.getByText('At /plans?year=2026')).toBeInTheDocument();
  });

  it('gives the page tooltips and messages, and a user-event instance', async () => {
    const { user } = renderWithProviders(<Page />);

    await user.click(screen.getByRole('button', { name: 'Say hello' }));

    expect(screen.getByRole('region', { name: 'Messages (F8)' })).toHaveTextContent('Hello');
  });

  it('has a stand-in sign-in page, so redirects can be followed', async () => {
    const { user, router } = renderWithProviders(<Page />);

    await user.click(screen.getByRole('link', { name: 'Sign in again' }));

    expect(router.state.location.pathname).toBe('/sign-in');
    expect(router.state.location.search).toBe('?next=%2Fplans');
    expect(screen.getByText('Sign-in page')).toBeInTheDocument();
  });

  it('starts signed out unless a person is given', () => {
    const signedOut = renderWithProviders(<Page />);
    expect(signedOut.queryClient.getQueryData(['session'])).toBeNull();

    const signedIn = renderWithProviders(<Page />, { user: PRIYA });
    expect(signedIn.queryClient.getQueryData(['session'])).toEqual(PRIYA);
  });

  it('gives the page the session, without asking the API', () => {
    const api = mockApi({});

    renderWithProviders(<SessionName />, { user: PRIYA });

    expect(screen.getByText('Signed in: Priya Raman')).toBeInTheDocument();
    expect(api.requests).toEqual([]);
  });

  it('can show the page at its own path, with a stand-in for every other address', async () => {
    const { router } = renderWithProviders(<Page />, {
      route: '/sign-in?next=%2Fplans',
      path: '/sign-in',
    });
    expect(screen.getByText('At /sign-in?next=%2Fplans')).toBeInTheDocument();

    await act(() => router.navigate('/plans'));

    expect(screen.getByText('Another page')).toBeInTheDocument();
  });

  it('can start on a location with router state', () => {
    renderWithProviders(<RouterState />, {
      route: { pathname: '/sign-in', state: { notice: 'signedOut' } },
      path: '/sign-in',
    });

    expect(screen.getByText('State: {"notice":"signedOut"}')).toBeInTheDocument();
  });
});

describe('renderRoutes', () => {
  it('renders the given routes, which ask GET /api/v1/me when no person is given', async () => {
    const api = mockApi({ 'GET /api/v1/me': { body: PRIYA } });

    renderRoutes([
      {
        path: '*',
        element: (
          <SessionProvider>
            <SessionName />
          </SessionProvider>
        ),
      },
    ]);

    expect(await screen.findByText('Signed in: Priya Raman')).toBeInTheDocument();
    expect(api.requests).toHaveLength(1);
  });
});

describe('findMessages', () => {
  it('finds the messages area once a message in it can be clicked', async () => {
    const { user } = renderWithProviders(<SaveLater />);
    await user.click(screen.getByRole('button', { name: 'Save' }));

    const messages = await findMessages();
    expect(messages).toHaveTextContent('Not saved');
    await user.click(within(messages).getByRole('button', { name: 'Try again' }));

    expect(screen.getByText('Tried again')).toBeInTheDocument();
  });
});
