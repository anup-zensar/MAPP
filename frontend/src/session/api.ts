import { useQuery } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { PreferencesUpdate } from './useSession';

export { setForbiddenHandler, setSessionLostHandler } from '@/api/client';

/** The signed-in person, or ApiError `signInRequired` / `sessionExpired` when nobody is. */
export function fetchMe() {
  return call(api.GET('/api/v1/me'));
}

export function postSignIn(email: string, password: string) {
  return call(api.POST('/api/v1/auth/sign-in', { body: { email, password } }));
}

export function postSignOut() {
  return call(api.POST('/api/v1/auth/sign-out'));
}

/** Saves only the preferences in `patch`. */
export function patchPreferences(patch: PreferencesUpdate) {
  return call(api.PATCH('/api/v1/me/preferences', { body: patch }));
}

/** Whether this environment offers demo sign-in, which also opens the developer pages. */
export function useEnvironment() {
  return useQuery({
    queryKey: ['environment'],
    queryFn: () => call(api.GET('/api/v1/environment')),
  });
}
