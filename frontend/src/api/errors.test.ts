import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { mockApi } from '@/test/mockApi';
import { api, call, setSessionLostHandler } from './client';
import { ApiError, conflictMessage, resolvedMessage } from './errors';
import type { components } from './schema';

type ErrorBody = components['schemas']['Error'];
type ErrorDetail = components['schemas']['ErrorDetail'];

const UNREACHABLE =
  "MAPP can't reach its server right now. Check that it is running and try again.";

function errorBody(
  code: ErrorDetail['code'],
  message: string,
  extra: Partial<ErrorDetail> = {},
): ErrorBody {
  return { error: { code, message, fields: [], ...extra } };
}

describe('call()', () => {
  it('returns the data of a successful answer', async () => {
    const environment = { environment: 'local', demoSignIn: true };
    mockApi({ 'GET /api/v1/environment': { body: environment } });

    await expect(call(api.GET('/api/v1/environment'))).resolves.toEqual(environment);
  });

  it('returns undefined for a 204 answer', async () => {
    mockApi({ 'POST /api/v1/auth/sign-out': { status: 204 } });

    await expect(call(api.POST('/api/v1/auth/sign-out'))).resolves.toBeUndefined();
  });

  it('treats an empty 500 answer as a server that cannot be reached (FR-020)', async () => {
    mockApi({ 'GET /api/v1/environment': { status: 500 } });

    await expect(call(api.GET('/api/v1/environment'))).rejects.toMatchObject({
      kind: 'unreachable',
      message: UNREACHABLE,
      status: 500,
    });
  });

  it('treats a text/plain 502 from the development proxy as unreachable (FR-020)', async () => {
    mockApi({
      'GET /api/v1/environment': () =>
        new Response('Bad gateway', { status: 502, headers: { 'Content-Type': 'text/plain' } }),
    });

    await expect(call(api.GET('/api/v1/environment'))).rejects.toMatchObject({
      kind: 'unreachable',
      message: UNREACHABLE,
    });
  });

  it('keeps the plain-language message of a JSON 500 error (FR-020)', async () => {
    const message = 'Something went wrong on our side. Please try again.';
    mockApi({
      'GET /api/v1/environment': { status: 500, body: errorBody('internal_error', message) },
    });

    await expect(call(api.GET('/api/v1/environment'))).rejects.toMatchObject({
      kind: 'other',
      message,
      status: 500,
    });
  });

  it('recognises a JSON 401 session_expired (FR-006)', async () => {
    mockApi({
      'GET /api/v1/me': { status: 401, body: errorBody('session_expired', 'Please sign in again') },
    });

    const failure = call(api.GET('/api/v1/me'));

    await expect(failure).rejects.toBeInstanceOf(ApiError);
    await expect(failure).rejects.toMatchObject({
      kind: 'sessionExpired',
      message: 'Please sign in again',
      status: 401,
    });
  });

  it('treats a network failure (TypeError) as unreachable (FR-020)', async () => {
    mockApi({ 'GET /api/v1/environment': () => Promise.reject(new TypeError('Failed to fetch')) });

    await expect(call(api.GET('/api/v1/environment'))).rejects.toMatchObject({
      kind: 'unreachable',
      message: UNREACHABLE,
      status: 0,
    });
  });

  it('passes a cancelled request (AbortError) on unchanged', async () => {
    const abort = new DOMException('The operation was aborted.', 'AbortError');
    mockApi({ 'GET /api/v1/environment': () => Promise.reject(abort) });

    await expect(call(api.GET('/api/v1/environment'))).rejects.toBe(abort);
  });

  it('shows a general message for a failed answer that is not MAPP JSON (FR-020)', async () => {
    mockApi({ 'GET /api/v1/environment': () => new Response('Not here', { status: 404 }) });

    await expect(call(api.GET('/api/v1/environment'))).rejects.toMatchObject({
      kind: 'other',
      message: 'Something went wrong. Please try again.',
      status: 404,
    });
  });

  it('carries every field problem at once (FR-020)', async () => {
    const fields = [
      { field: 'email', message: 'Enter your e-mail' },
      { field: 'password', message: 'Enter the password' },
    ];
    mockApi({
      'POST /api/v1/auth/sign-in': {
        status: 400,
        body: errorBody('validation_failed', 'Some fields need your attention.', { fields }),
      },
    });

    await expect(
      call(api.POST('/api/v1/auth/sign-in', { body: { email: '', password: '' } })),
    ).rejects.toMatchObject({ kind: 'validation', fields });
  });

  it.each([
    ['validation_failed', 400, 'validation'],
    ['bad_request', 400, 'badRequest'],
    ['sign_in_required', 401, 'signInRequired'],
    ['session_expired', 401, 'sessionExpired'],
    ['incorrect_credentials', 401, 'incorrectCredentials'],
    ['not_found', 404, 'notFound'],
    ['version_conflict', 409, 'conflict'],
    ['restore_parent_first', 409, 'restoreParentFirst'],
    ['undo_expired', 409, 'undoExpired'],
    ['already_resolved', 409, 'alreadyResolved'],
    ['name_taken', 409, 'nameTaken'],
    ['in_use', 409, 'inUse'],
    ['cannot_restore', 409, 'cannotRestore'],
    ['not_allowed', 409, 'notAllowed'],
    ['not_ready', 409, 'notReady'],
    ['parent_deleted_permanently', 409, 'parentDeletedPermanently'],
    ['internal_error', 500, 'other'],
  ] as const)('turns the code %s (%i) into the kind %s', async (code, status, kind) => {
    mockApi({
      'POST /api/v1/recycle-bin/BIN-1001/undo': { status, body: errorBody(code, 'A message.') },
    });

    await expect(
      call(
        api.POST('/api/v1/recycle-bin/{entryId}/undo', {
          params: { path: { entryId: 'BIN-1001' } },
        }),
      ),
    ).rejects.toMatchObject({ kind, message: 'A message.', status });
  });
});

describe('ApiError.missing (feature 004, not_ready)', () => {
  it('carries the needed fields grouped by record, and is empty for other errors', () => {
    const missing = [
      {
        record: { type: 'plan', id: 'P-1001', name: 'FY2026 Lubricants Europe' },
        fields: [{ field: 'ownerId', label: 'Owner' }],
      },
    ];
    const notReady = ApiError.fromBody(
      409,
      errorBody('not_ready', 'Publish needs 1 field', { missing }),
    );

    expect(notReady).toMatchObject({ kind: 'notReady', missing });
    expect(new ApiError('other').missing).toEqual([]);
  });
});

describe('the session-lost hook', () => {
  const onSessionLost = vi.fn();

  beforeEach(() => {
    setSessionLostHandler(onSessionLost);
  });

  afterEach(() => {
    setSessionLostHandler(null);
    onSessionLost.mockReset();
  });

  it('ends the session when a signed-in call answers sign_in_required (FR-006)', async () => {
    mockApi({
      'GET /api/v1/recycle-bin': {
        status: 401,
        body: errorBody('sign_in_required', 'Please sign in to continue.'),
      },
    });

    await expect(call(api.GET('/api/v1/recycle-bin'))).rejects.toMatchObject({
      kind: 'signInRequired',
    });
    expect(onSessionLost).toHaveBeenCalledTimes(1);
  });

  it('ends the session when a signed-in call answers session_expired (FR-006)', async () => {
    mockApi({
      'PATCH /api/v1/me/preferences': {
        status: 401,
        body: errorBody('session_expired', 'Please sign in again'),
      },
    });

    await expect(
      call(api.PATCH('/api/v1/me/preferences', { body: { theme: 'dark' } })),
    ).rejects.toMatchObject({ kind: 'sessionExpired' });
    expect(onSessionLost).toHaveBeenCalledTimes(1);
  });

  it('leaves the session check and the sign-in calls to the session and the sign-in page', async () => {
    mockApi({
      'GET /api/v1/me': { status: 401, body: errorBody('session_expired', 'Please sign in again') },
      'POST /api/v1/auth/sign-in': {
        status: 401,
        body: errorBody('incorrect_credentials', 'Incorrect e-mail or password.'),
      },
    });

    await expect(call(api.GET('/api/v1/me'))).rejects.toBeInstanceOf(ApiError);
    await expect(
      call(api.POST('/api/v1/auth/sign-in', { body: { email: 'a@b.c', password: 'x' } })),
    ).rejects.toBeInstanceOf(ApiError);
    expect(onSessionLost).not.toHaveBeenCalled();
  });

  it('ignores failures that are not about the session', async () => {
    mockApi({
      'GET /api/v1/recycle-bin': { status: 404, body: errorBody('not_found', 'Not here.') },
    });

    await expect(call(api.GET('/api/v1/recycle-bin'))).rejects.toMatchObject({ kind: 'notFound' });
    expect(onSessionLost).not.toHaveBeenCalled();
  });
});

describe('messages with a time', () => {
  // "Now" is 12 Mar 2026, 09:00 in the user's time zone (UTC in tests).
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-03-12T09:00:00Z'));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  function conflictError(changedAt: string): ApiError {
    return ApiError.fromBody(
      409,
      errorBody('version_conflict', 'Not saved — Daniel Okoro changed this sample folder', {
        conflict: { changedBy: 'Daniel Okoro', changedAt },
      }),
    );
  }

  function resolvedError(how: 'undone' | 'restored' | 'deletedPermanently', at: string): ApiError {
    return ApiError.fromBody(
      409,
      errorBody('already_resolved', "'Ideas' is no longer in the recycle bin.", {
        resolution: { how, by: { id: 'U-1002', name: 'Priya Raman' }, at },
      }),
    );
  }

  it('says who changed a record and at what time today (FR-029)', () => {
    expect(conflictMessage(conflictError('2026-03-12T08:32:00Z'), 'sample folder')).toBe(
      'Not saved — Daniel Okoro changed this sample folder at 08:32',
    );
  });

  it('adds the date when the change was not today (FR-029)', () => {
    expect(conflictMessage(conflictError('2026-03-11T14:32:00Z'), 'sample folder')).toBe(
      'Not saved — Daniel Okoro changed this sample folder on 11 Mar 2026 at 14:32',
    );
  });

  it('says who restored an item and when (FR-025)', () => {
    expect(resolvedMessage(resolvedError('restored', '2026-03-12T08:05:00Z'), 'Ideas')).toBe(
      "'Ideas' is no longer in the recycle bin — Priya Raman restored it at 08:05",
    );
    expect(resolvedMessage(resolvedError('restored', '2026-03-10T16:45:00Z'), 'Ideas')).toBe(
      "'Ideas' is no longer in the recycle bin — Priya Raman restored it on 10 Mar 2026 at 16:45",
    );
  });

  it('says who undid a delete and when (FR-025)', () => {
    expect(resolvedMessage(resolvedError('undone', '2026-03-12T08:05:00Z'), 'Ideas')).toBe(
      "'Ideas' is no longer in the recycle bin — Priya Raman undid the delete at 08:05",
    );
    expect(resolvedMessage(resolvedError('undone', '2026-03-10T16:45:00Z'), 'Ideas')).toBe(
      "'Ideas' is no longer in the recycle bin — Priya Raman undid the delete on 10 Mar 2026 at 16:45",
    );
  });

  it('says who deleted an item permanently and when (feature 011, FR-032)', () => {
    expect(
      resolvedMessage(resolvedError('deletedPermanently', '2026-03-12T08:05:00Z'), 'Ideas'),
    ).toBe("'Ideas' is no longer in the recycle bin — Priya Raman deleted it permanently at 08:05");
  });

  it('writes the date in the person’s date format (feature 003, FR-024)', () => {
    expect(conflictMessage(conflictError('2026-03-11T14:32:00Z'), 'person', 'DD/MM/YYYY')).toBe(
      'Not saved — Daniel Okoro changed this person on 11/03/2026 at 14:32',
    );
    expect(
      resolvedMessage(resolvedError('restored', '2026-03-10T16:45:00Z'), 'Ideas', 'YYYY-MM-DD'),
    ).toBe(
      "'Ideas' is no longer in the recycle bin — Priya Raman restored it on 2026-03-10 at 16:45",
    );
  });

  it('falls back to the API’s message when the details are missing', () => {
    const conflict = ApiError.fromBody(409, errorBody('version_conflict', 'Not saved — someone'));
    const resolved = ApiError.fromBody(409, errorBody('already_resolved', 'Gone.'));

    expect(conflictMessage(conflict, 'sample folder')).toBe('Not saved — someone');
    expect(resolvedMessage(resolved, 'Ideas')).toBe('Gone.');
  });
});
