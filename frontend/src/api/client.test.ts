import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { mockApi } from '@/testing/mock-api';
import { apiError, reply } from '@/testing/replies';
import { api, call, setSessionLostHandler } from './client';
import { ApiError } from './errors';

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
    mockApi([
      reply('GET /api/v1/recycle-bin', {
        status: 401,
        body: apiError('sign_in_required', 'Please sign in to continue.'),
      }),
    ]);

    await expect(call(api.GET('/api/v1/recycle-bin'))).rejects.toMatchObject({
      kind: 'signInRequired',
    });
    expect(onSessionLost).toHaveBeenCalledTimes(1);
  });

  it('ends the session when a signed-in call answers session_expired (FR-006)', async () => {
    mockApi([
      reply('PATCH /api/v1/me/preferences', {
        status: 401,
        body: apiError('session_expired', 'Please sign in again'),
      }),
    ]);

    await expect(
      call(api.PATCH('/api/v1/me/preferences', { body: { theme: 'dark' } })),
    ).rejects.toMatchObject({ kind: 'sessionExpired' });
    expect(onSessionLost).toHaveBeenCalledTimes(1);
  });

  it('leaves the session check and the sign-in calls to the session and the sign-in page', async () => {
    mockApi([
      reply('GET /api/v1/me', {
        status: 401,
        body: apiError('session_expired', 'Please sign in again'),
      }),
      reply('POST /api/v1/auth/sign-in', {
        status: 401,
        body: apiError('incorrect_credentials', 'Incorrect e-mail or password.'),
      }),
    ]);

    await expect(call(api.GET('/api/v1/me'))).rejects.toBeInstanceOf(ApiError);
    await expect(
      call(api.POST('/api/v1/auth/sign-in', { body: { email: 'a@b.c', password: 'x' } })),
    ).rejects.toBeInstanceOf(ApiError);
    expect(onSessionLost).not.toHaveBeenCalled();
  });

  it('ignores failures that are not about the session', async () => {
    mockApi([
      reply('GET /api/v1/recycle-bin', {
        status: 404,
        body: apiError('not_found', 'Not here.'),
      }),
    ]);

    await expect(call(api.GET('/api/v1/recycle-bin'))).rejects.toMatchObject({ kind: 'notFound' });
    expect(onSessionLost).not.toHaveBeenCalled();
  });
});
