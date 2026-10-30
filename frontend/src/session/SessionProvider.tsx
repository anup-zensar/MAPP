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

const CHECK_KEPT_FOR_MS = 30_000;

/** Knows who is signed in: asks GET /api/v1/me, ends sessions in one place, follows other tabs. */
export function SessionProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const location = useLocation();
  const { show } = useToast();
  const [visitorTheme, setVisitorTheme] = useState<ThemePreference | null>(null);

  // Set in a layout effect, which runs before other effects: a page just opened is the way back.
  const pageOnScreen = useRef<Path>(location);
  useLayoutEffect(() => {
    pageOnScreen.current = location;
  }, [location]);

  // A check of /me in flight when a session ended or began must not answer for the new session.
  const sessionChanges = useRef(0);

  const endSession = useCallback(
    (reason: SessionEnd) => {
      // Several answers can say "signed out" at the same time; the session ends only once.
      if (queryClient.getQueryData(SESSION_KEY) === null) {
        return;
      }
      // flushSync: forget the person and show sign-in together, so no page sees the old state.
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

  const followOtherPerson = useCallback(
    (person: CurrentUser) => {
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
        if (queryClient.getQueryData(SESSION_KEY) || error.kind === 'sessionExpired') {
          endSession('sessionLost');
        }
        return null;
      }
    },
    staleTime: CHECK_KEPT_FOR_MS,
  });
  const { data, isError, error, refetch: refetchSession } = sessionQuery;
  const person = data ?? null;

  const refetch = useCallback(() => {
    // An answer already on its way counts as the new check, so checks never pile up.
    refetchSession({ cancelRefetch: false });
  }, [refetchSession]);

  useEffect(() => {
    setSessionLostHandler(() => endSession('sessionLost'));
    return () => setSessionLostHandler(null);
  }, [endSession]);

  // Access may have changed since GET /api/v1/me: ask again so navigation and gates follow at once.
  useEffect(() => {
    setForbiddenHandler(refetch);
    return () => setForbiddenHandler(null);
  }, [refetch]);

  // React allows this state update while rendering; it happens at most once per sign-in.
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
      show({ message: messageOf(failure), action: tryAgain(signOut) });
      return;
    }
    endSession('signedOut');
  }

  function updatePreferences(patch: PreferencesUpdate) {
    const signedIn = queryClient.getQueryData<CurrentUser | null>(SESSION_KEY);
    if (!signedIn) {
      if (patch.theme) {
        setVisitorTheme(patch.theme);
      }
      return;
    }
    queryClient.setQueryData(SESSION_KEY, {
      ...signedIn,
      preferences: { ...signedIn.preferences, ...patch },
    });
    savePreferences(patch);
  }

  async function savePreferences(patch: PreferencesUpdate) {
    try {
      await patchPreferences(patch);
    } catch (failure) {
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

function sessionLostAddress(page: Path): string {
  const address = page.pathname + page.search + page.hash;
  return matchPath('/sign-in', page.pathname) ? address : signInAddress(address);
}

function isSessionGone(error: unknown): error is ApiError {
  return (
    error instanceof ApiError &&
    (error.kind === 'signInRequired' || error.kind === 'sessionExpired')
  );
}

function isOtherData(query: Query): boolean {
  return query.queryKey[0] !== SESSION_KEY[0];
}

function messageOf(failure: unknown): string {
  return (failure instanceof ApiError ? failure : new ApiError('other')).message;
}

function tryAgain(retry: () => void): ToastOptions['action'] {
  return { label: 'Try again', altText: 'Try again', onClick: retry };
}
