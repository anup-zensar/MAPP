import { describe, expect, it } from 'vitest';
import { mockApi, takeUnmockedRequests } from './mock-api';
import { ORIGIN, pending, reply } from './replies';

describe('mockApi', () => {
  it('answers a request with the canned JSON of its operation', async () => {
    mockApi([
      reply('GET /api/v1/environment', { body: { environment: 'local', demoSignIn: true } }),
    ]);

    const response = await fetch(`${ORIGIN}/api/v1/environment`);

    expect(response.status).toBe(200);
    expect(response.headers.get('Content-Type')).toBe('application/json');
    expect(await response.json()).toEqual({ environment: 'local', demoSignIn: true });
  });

  it('uses the given status and sends no body when there is none', async () => {
    mockApi([reply('POST /api/v1/auth/sign-out', { status: 204 })]);

    const response = await fetch(`${ORIGIN}/api/v1/auth/sign-out`, { method: 'POST' });

    expect(response.status).toBe(204);
    expect(await response.text()).toBe('');
  });

  it('answers only the given path parameters, and lets a later answer win', async () => {
    mockApi([
      reply('GET /api/v1/plans/{planId}', { status: 404 }),
      reply('GET /api/v1/plans/{planId}', { status: 410 }, { planId: 'P-1004' }),
    ]);

    expect((await fetch(`${ORIGIN}/api/v1/plans/P-1004`)).status).toBe(410);
    expect((await fetch(`${ORIGIN}/api/v1/plans/P-1005`)).status).toBe(404);
  });

  it('records every request with its query and JSON body', async () => {
    const api = mockApi([reply('PATCH /api/v1/me/preferences', { status: 204 })]);

    await fetch(`${ORIGIN}/api/v1/me/preferences?x=1`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ theme: 'dark' }),
    });

    expect(api.requests).toEqual([
      { method: 'PATCH', path: '/api/v1/me/preferences', search: '?x=1', body: { theme: 'dark' } },
    ]);
    expect(api.bodiesOf('PATCH /api/v1/me/preferences')).toEqual([{ theme: 'dark' }]);
  });

  it('lets a function answer, for responses in order, other content or failures', async () => {
    let calls = 0;
    mockApi([
      reply('GET /api/v1/me', () => (++calls === 1 ? { status: 401 } : { status: 403 })),
      reply('GET /api/v1/environment', () => new Response('Bad gateway', { status: 502 })),
      reply('GET /api/v1/recycle-bin', () => Promise.reject(new TypeError('Failed to fetch'))),
      reply('GET /api/v1/notifications', pending),
    ]);

    expect((await fetch(`${ORIGIN}/api/v1/me`)).status).toBe(401);
    expect((await fetch(`${ORIGIN}/api/v1/me`)).status).toBe(403);
    expect(await (await fetch(`${ORIGIN}/api/v1/environment`)).text()).toBe('Bad gateway');
    await expect(fetch(`${ORIGIN}/api/v1/recycle-bin`)).rejects.toThrow();
  });

  it('remembers requests without an answer, so the test setup can fail the test', async () => {
    mockApi([]);

    const response = await fetch(`${ORIGIN}/api/v1/sample-folders`);

    expect(response.status).toBe(501);
    // Taking them here keeps this test green; normally the setup's afterEach reports them.
    expect(takeUnmockedRequests()).toEqual(['GET /api/v1/sample-folders']);
    expect(takeUnmockedRequests()).toEqual([]);
  });

  it('answers the app frame’s background requests unless the test answers them itself', async () => {
    mockApi([reply('GET /api/v1/notifications', { body: { items: [], total: 3, unread: 3 } })]);

    const summary = await fetch(`${ORIGIN}/api/v1/approvals/summary`);
    const notifications = await fetch(`${ORIGIN}/api/v1/notifications`);

    expect(await summary.json()).toEqual({
      waitingForMe: 0,
      allOpen: 0,
      approvedThisMonth: 0,
      declinedThisMonth: 0,
    });
    expect(await notifications.json()).toEqual({ items: [], total: 3, unread: 3 });
    expect(takeUnmockedRequests()).toEqual([]);
  });
});
