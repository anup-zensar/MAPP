import { useQuery, useQueryClient } from '@tanstack/react-query';
import type { Query } from '@tanstack/react-query';
import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react';
import type { ReactNode } from 'react';
import { matchPath, useLocation, useNavigate } from 'react-router';
import type { Path } from 'react-router';
import { ApiError } from '@/api/errors';
import { useToast } from '@/ui/Toast/useToast';
import type { ToastOptions } from '@/ui/Toast/useToast';
import { readStoredPreference } from '@/ui/theme';
import type { ThemePreference } from '@/ui/theme';
import { useTheme } from '@/ui/useTheme';
import {
  fetchMe,
  patchPreferences,
  postSignIn,
  postSignOut,
  setForbiddenHandler,
  setSessionLostHandler,
} from './api';
import { unsavedFormTitle } from './lost-changes';
import { SESSION_KEY, SessionContext, signInAddress } from './useSession';
import type { CurrentUser, PreferencesUpdate, SessionEnd, SessionStatus } from './useSession';

/** A check younger than this is not repeated when the window regains focus (research R15). */
const CHECK_KEPT_FOR_MS = 30_000;

/**
 * Knows who is signed in and keeps the whole app in step with it (plan "Session", research R15
 * and R16): it asks GET /api/v1/me, ends a session in one place, notices when another person signed
 * in in another tab of this browser, and shows the person's light or dark mode.
 */
export function SessionProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const location = useLocation();
  const { show } = useToast();
  // The light or dark mode chosen with the theme switch while nobody is signed in (research R16).
  // null: no choice since the last person was signed in.
  const [visitorTheme, setVisitorTheme] = useState<ThemePreference | null>(null);

  // The page on screen, for endSession. A layout effect runs before every other effect, so a
  // check started by a page that has just opened already sends people back to that page.
  const pageOnScreen = useRef<Path>(location);
  useLayoutEffect(() => {
    pageOnScreen.current = location;
  }, [location]);

  // Counts the sessions this tab has ended or begun. A check of GET /api/v1/me that was on its way
  // when the session ended (or a new one began) answers for a session that is no longer the one
  // on screen: its answer must not bring the person back after signing out, nor end a new session.
  const sessionChanges = useRef(0);

  /**
   * The one way a session ends in the web app (plan "Session"): the sign-in page opens with "You
   * have been signed out." or, when the session was lost, with "Please sign in again" and the
   * way back to the page that was on screen.
   */
  const endSession = useCallback(
    (reason: SessionEnd) => {
      // Several answers can say "signed out" at the same time; the session ends only once.
      if (queryClient.getQueryData(SESSION_KEY) === null) {
        return;
      }
      // First forget the person, then show the sign-in page in the same moment (flushSync).
      // Otherwise the page on screen, or the sign-in page, would see the old state in between
      // and send people on a second time, without the notice.
      // The form whose typed changes are lost is named on the sign-in page (038 FR-031).
      const lostChanges = reason === 'sessionLost' ? unsavedFormTitle() : null;
      sessionChanges.current += 1;
      queryClient.setQueryData(SESSION_KEY, null);
      const to = reason === 'sessionLost' ? sessionLostAddress(pageOnScreen.current) : '/sign-in';
      const state = lostChanges ? { notice: reason, lostChanges } : { notice: reason };
      navigate(to, { replace: true, state, flushSync: true });
      // Nothing of the person's data stays in the browser, so Back cannot show it again.
      queryClient.removeQueries({ predicate: isOtherData });
    },
    [queryClient, navigate],
  );

  /** Another person signed in in another tab of this browser (spec edge case). */
  const followOtherPerson = useCallback(
    (person: CurrentUser) => {
      // Nothing may be done in the first person's name: forget their data and start on Home.
      queryClient.removeQueries({ predicate: isOtherData });
      queryClient.setQueryData(SESSION_KEY, person);
      navigate('/');
      show({
        message: `You're now signed in as ${person.name} — this was changed in another tab.`,
      });
    },
    [queryClient, navigate, show],
  );

  const sessionQuery = useQuery({
    queryKey: SESSION_KEY,
    queryFn: async (): Promise<CurrentUser | null> => {
      const changesBefore = sessionChanges.current;
      // The session ended or began while this check was on its way: keep what is on screen now.
      const outOfDate = () => sessionChanges.current !== changesBefore;
      try {
        const person = await fetchMe();
        if (outOfDate()) {
          return queryClient.getQueryData<CurrentUser | null>(SESSION_KEY) ?? null;
        }
        const known = queryClient.getQueryData<CurrentUser | null>(SESSION_KEY);
        if (known && known.id !== person.id) {
          followOtherPerson(person);
        }
        return person;
      } catch (error) {
        if (!isSessionGone(error)) {
          throw error;
        }
        if (outOfDate()) {
          return queryClient.getQueryData<CurrentUser | null>(SESSION_KEY) ?? null;
        }
        // The person's session ended (it expired, was signed out in another tab, or the person
        // was deactivated), or this browser came back after its session expired.
        if (queryClient.getQueryData(SESSION_KEY) || error.kind === 'sessionExpired') {
          endSession('sessionLost');
        }
        return null;
      }
    },
    staleTime: CHECK_KEPT_FOR_MS,
  });
  // data: the person, null when nobody is signed in, undefined before the first answer.
  const { data, isError, error, refetch: refetchSession } = sessionQuery;
  const person = data ?? null;

  const refetch = useCallback(() => {
    // An answer already on its way counts as the new check, so checks never pile up.
    refetchSession({ cancelRefetch: false });
  }, [refetchSession]);

  // Any other API call that finds the session gone ends it here (plan "API client and errors").
  useEffect(() => {
    setSessionLostHandler(() => endSession('sessionLost'));
    return () => setSessionLostHandler(null);
  }, [endSession]);

  // A call refused by the person's roles: their access may have changed since GET /api/v1/me, so it is
  // asked again and navigation and page gates follow at once (feature 014, FR-020).
  useEffect(() => {
    setForbiddenHandler(refetch);
    return () => setForbiddenHandler(null);
  }, [refetch]);

  // While someone is signed in, their saved choice counts and a visitor's earlier choice is
  // forgotten. A signed-out page then shows this browser's last theme — after signing out, the
  // person's own — just as a reload would. (React allows this kind of state update while
  // rendering; it happens at most once per sign-in.)
  if (person && visitorTheme !== null) {
    setVisitorTheme(null);
  }
  const theme = useTheme(
    person ? person.preferences.theme : (visitorTheme ?? readStoredPreference()),
  );

  async function signIn(email: string, password: string) {
    const signedIn = await postSignIn(email, password);
    sessionChanges.current += 1;
    queryClient.setQueryData(SESSION_KEY, signedIn);
  }

  async function signOut() {
    try {
      await postSignOut();
    } catch (failure) {
      // The person is still signed in: say why in plain words, with Try again (FR-020).
      show({ message: messageOf(failure), action: tryAgain(signOut) });
      return;
    }
    endSession('signedOut');
  }

  function updatePreferences(patch: PreferencesUpdate) {
    const signedIn = queryClient.getQueryData<CurrentUser | null>(SESSION_KEY);
    if (!signedIn) {
      // Nobody is signed in: the theme switch changes only this browser's choice (research R16).
      if (patch.theme) {
        setVisitorTheme(patch.theme);
      }
      return;
    }
    // The screen follows the choice at once; it is saved in the background (research R16).
    queryClient.setQueryData(SESSION_KEY, {
      ...signedIn,
      preferences: { ...signedIn.preferences, ...patch },
    });
    savePreferences(patch);
  }

  async function savePreferences(patch: PreferencesUpdate) {
    try {
      // Only the changed preference is sent.
      await patchPreferences(patch);
    } catch (failure) {
      // A session that has ended is handled by endSession alone (plan "Preferences and theme").
      if (!isSessionGone(failure)) {
        show({
          message: "Your choice couldn't be saved.",
          action: tryAgain(() => savePreferences(patch)),
        });
      }
    }
  }

  const status = sessionStatus(data, isError);

  return (
    <SessionContext
      value={{
        status,
        user: person,
        error: status === 'unreachable' ? error : null,
        theme,
        refetch,
        signIn,
        signOut,
        updatePreferences,
      }}
    >
      {children}
    </SessionContext>
  );
}

/**
 * The session's state after the last check (plan "Session"): a 401 means signed out; any other
 * failure while nobody is known means unreachable.
 */
function sessionStatus(data: CurrentUser | null | undefined, checkFailed: boolean): SessionStatus {
  if (data) {
    // A failed check while someone is signed in changes nothing.
    return 'signedIn';
  }
  if (checkFailed) {
    return 'unreachable';
  }
  return data === null ? 'signedOut' : 'loading';
}

/**
 * Where a lost session sends people: the sign-in page, with the page on screen as the way back
 * (FR-005). A session found expired on the sign-in page itself keeps that page's own address,
 * and so its own way back.
 */
function sessionLostAddress(page: Path): string {
  const address = page.pathname + page.search + page.hash;
  return matchPath('/sign-in', page.pathname) ? address : signInAddress(address);
}

/** An answer that says there is no session (any more): the API's two "sign in" codes. */
function isSessionGone(error: unknown): error is ApiError {
  return (
    error instanceof ApiError &&
    (error.kind === 'signInRequired' || error.kind === 'sessionExpired')
  );
}

/** Every cached answer except the session itself. */
function isOtherData(query: Query): boolean {
  return query.queryKey[0] !== SESSION_KEY[0];
}

/** The plain-language message of a failed API call (FR-020). */
function messageOf(failure: unknown): string {
  return (failure instanceof ApiError ? failure : new ApiError('other')).message;
}

/** The Try again button of a message (FR-020). */
function tryAgain(retry: () => void): ToastOptions['action'] {
  return { label: 'Try again', altText: 'Try again', onClick: retry };
}
