import { Outlet, ScrollRestoration } from 'react-router';
import { SessionProvider } from '@/session/SessionProvider';

/**
 * The root of every page (plan "Router"): the session around the whole app, the scroll position
 * restored when people go Back or Forward, and the page of the current address.
 */
export function RootLayout() {
  return (
    <SessionProvider>
      <ScrollRestoration />
      <Outlet />
    </SessionProvider>
  );
}
