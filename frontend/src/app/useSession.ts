import { createContext, useContext } from 'react';
import type { components } from '@/api/schema';
import type { Theme } from './theme';

/** The signed-in person with their preferences, as GET /api/me answers. */
export type CurrentUser = components['schemas']['CurrentUser'];

/** The preferences to change; a field that is left out stays as it is. */
export type PreferencesUpdate = components['schemas']['PreferencesUpdate'];

/**
 * Where TanStack Query keeps the session: the person GET /api/me answered, null when nobody is
 * signed in, and nothing at all until the first answer.
 */
export const SESSION_KEY = ['session'];

/**
 * - loading: the first check has not answered yet;
 * - signedIn: someone is signed in;
 * - signedOut: nobody is signed in;
 * - unreachable: nobody is known and the API could not be asked (research R15).
 */
export type SessionStatus = 'loading' | 'signedIn' | 'signedOut' | 'unreachable';

/**
 * Why a session ended: the person chose Sign out, or the API said the session is gone (it
 * expired, or it ended in another tab). The sign-in page shows the matching notice.
 */
export type SessionEnd = 'signedOut' | 'sessionLost';

/** What useSession() gives every part of the app. */
export interface Session {
  status: SessionStatus;
  /** The signed-in person; null when nobody is signed in or nobody is known yet. */
  user: CurrentUser | null;
  /** Why the session could not be checked (status unreachable); null otherwise. */
  error: Error | null;
  /** The colours on screen, for the theme switch. */
  theme: Theme;
  /** Asks the API again who is signed in; an answer already on its way counts as that check. */
  refetch: () => void;
  /** Signs in with an e-mail and the demo password; throws an ApiError when that fails. */
  signIn: (email: string, password: string) => Promise<void>;
  /**
   * Signs out and opens the sign-in page with "You have been signed out.". When that fails, the
   * person stays signed in and a message says why, with Try again.
   */
  signOut: () => Promise<void>;
  /**
   * Shows the changed preferences at once and saves them in the background (research R16).
   * When nobody is signed in, only the theme changes, and only in this browser.
   */
  updatePreferences: (patch: PreferencesUpdate) => void;
}

/** Shared by SessionProvider (which provides it) and useSession (which reads it). */
export const SessionContext = createContext<Session | null>(null);

/** Who is signed in, and how to sign in, sign out or change preferences (plan "Session"). */
export function useSession(): Session {
  const session = useContext(SessionContext);
  if (!session) {
    throw new Error('useSession needs the SessionProvider around the app (RootLayout.tsx).');
  }
  return session;
}

/**
 * The signed-in person, for the app layout and the pages inside RequireSignIn: they are only
 * shown when someone is signed in.
 */
export function useCurrentUser(): CurrentUser {
  const { user } = useSession();
  if (!user) {
    throw new Error('useCurrentUser only works inside RequireSignIn, where someone is signed in.');
  }
  return user;
}

/** The address of the sign-in page that brings people back to `next` afterwards (FR-005). */
export function signInAddress(next: string): string {
  return `/sign-in?${new URLSearchParams({ next })}`;
}
