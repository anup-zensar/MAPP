/**
 * The web app's only way to talk to MAPP's API (research R11, R15).
 *
 * `api` is typed from the contract (backend/api/openapi.yaml → generated/schema.d.ts), so paths,
 * parameters and bodies are checked by TypeScript. Wrap every call in call(), which returns the data
 * or throws an ApiError:
 *
 *   const me = await call(api.GET('/api/v1/me'));
 */
import createClient from 'openapi-fetch';
import { ApiError } from './errors';
import type { components, paths } from './generated/schema';

type ErrorBody = components['schemas']['Error'];

export const api = createClient<paths>({
  // An absolute address, because the test environment cannot build requests from a relative one.
  baseUrl: window.location.origin,
  // openapi-fetch keeps the fetch it is given; looking it up on every call lets tests replace it.
  fetch: (request) => globalThis.fetch(request),
});

/** What openapi-fetch gives back for a call: the data, or the error, and the answer itself. */
interface ApiResult<T> {
  data?: T;
  error?: unknown;
  response: Response;
}

/**
 * Waits for an API call and returns its data (undefined for a 204 answer), or throws an ApiError:
 * - no answer at all (a network failure) → unreachable;
 * - an error answer from the API → its code and plain-language message;
 * - any other failed answer → unreachable for 5xx (the development server answers that way while
 *   the API is stopped), otherwise other.
 * A cancelled request (AbortError) is passed on unchanged, so cancelling never shows an error.
 */
export async function call<T>(request: Promise<ApiResult<T>>): Promise<T> {
  let result: ApiResult<T>;
  try {
    result = await request;
  } catch (error) {
    if (isAbortError(error)) {
      throw error;
    }
    throw new ApiError('unreachable');
  }

  const { data, error, response } = result;
  if (response.ok) {
    // eslint-disable-next-line @typescript-eslint/consistent-type-assertions -- the one place that types a response body (F5)
    return data as T;
  }
  if (isErrorBody(error)) {
    throw ApiError.fromBody(response.status, error);
  }
  throw new ApiError(response.status >= 500 ? 'unreachable' : 'other', {
    status: response.status,
  });
}

let sessionLostHandler: (() => void) | null = null;

/**
 * Tells the client what to do when a signed-in call finds that the session has ended — expired,
 * signed out in another tab, or the person deactivated. The session provider sets it; null
 * removes it.
 */
export function setSessionLostHandler(handler: (() => void) | null): void {
  sessionLostHandler = handler;
}

let forbiddenHandler: (() => void) | null = null;

/**
 * Tells the client what to do when a call is refused because of the person's roles (403 forbidden,
 * feature 014): the session provider reloads the person's access, so the screens follow a role
 * that changed while a page was open (FR-020). null removes it.
 */
export function setForbiddenHandler(handler: (() => void) | null): void {
  forbiddenHandler = handler;
}

// Any call answering "sign in again" means the session has ended — except signing in itself and
// the session check (GET /api/v1/me), whose 401 answers the session code handles (research R15). Any
// call refused by the person's roles means their access may have changed (feature 014, FR-020).
api.use({
  async onResponse({ request, response }) {
    const sessionLost = response.status === 401 && !isSessionCall(request);
    if (!sessionLost && response.status !== 403) {
      return;
    }
    const body: unknown = await response
      .clone()
      .json()
      .catch(() => null);
    if (!isErrorBody(body)) {
      return;
    }
    const { code } = body.error;
    if (sessionLost && (code === 'sign_in_required' || code === 'session_expired')) {
      sessionLostHandler?.();
    } else if (response.status === 403 && code === 'forbidden') {
      forbiddenHandler?.();
    }
  },
});

function isSessionCall(request: Request): boolean {
  const path = new URL(request.url).pathname;
  return path.startsWith('/api/v1/auth/') || (request.method === 'GET' && path === '/api/v1/me');
}

/** An error answer of MAPP's API: JSON {"error": {"code", …}} (anything else came from elsewhere). */
function isErrorBody(value: unknown): value is ErrorBody {
  if (typeof value !== 'object' || value === null || !('error' in value)) {
    return false;
  }
  const { error } = value;
  return (
    typeof error === 'object' && error !== null && 'code' in error && typeof error.code === 'string'
  );
}

/**
 * A request cancelled on purpose. Checked by its name, because the error can be the DOMException
 * of another window, which instanceof would not recognise.
 */
function isAbortError(error: unknown): boolean {
  return (
    typeof error === 'object' && error !== null && 'name' in error && error.name === 'AbortError'
  );
}
