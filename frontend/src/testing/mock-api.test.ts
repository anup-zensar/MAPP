import { describe, expect, it } from 'vitest';
import { FRAME_ANSWERS, mockApi, takeUnmockedRequests } from './mock-api';

const ORIGIN = window.location.origin;

describe('mockApi', () => {
  it('answers a request with the canned JSON for its "METHOD /path"', async () => {
    mockApi({ 'GET /api/v1/environment': { body: { status: 'ok' } } });

    const response = await fetch(`${ORIGIN}/api/v1/environment`);

    expect(response.status).toBe(200);
    expect(response.headers.get('Content-Type')).toBe('application/json');
    expect(await response.json()).toEqual({ status: 'ok' });
  });

  it('uses the given status and sends no body when there is none', async () => {
    mockApi({ 'POST /api/v1/auth/sign-out': { status: 204 } });

    const response = await fetch(`${ORIGIN}/api/v1/auth/sign-out`, { method: 'POST' });

    expect(response.status).toBe(204);
    expect(await response.text()).toBe('');
  });

  it('matches on the path only, so the query string does not matter', async () => {
    const api = mockApi({ 'GET /api/v1/activity-log': { body: { items: [], total: 0 } } });

    const response = await fetch(`${ORIGIN}/api/v1/activity-log?limit=20`);

    expect(response.status).toBe(200);
    expect(api.requests).toMatchObject([
      { method: 'GET', path: '/api/v1/activity-log', search: '?limit=20' },
    ]);
  });

  it('records every request with its JSON body', async () => {
    const api = mockApi({ 'PATCH /api/v1/me/preferences': { body: { theme: 'dark' } } });

    await fetch(`${ORIGIN}/api/v1/me/preferences`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ theme: 'dark' }),
    });

    expect(api.requests).toEqual([
      { method: 'PATCH', path: '/api/v1/me/preferences', search: '', body: { theme: 'dark' } },
    ]);
    expect(api.bodiesOf('PATCH /api/v1/me/preferences')).toEqual([{ theme: 'dark' }]);
  });

  it('lets a function answer, for responses in order, slow answers or failures', async () => {
    let calls = 0;
    mockApi({
      'GET /api/v1/me': () => (++calls === 1 ? { body: { id: 'U-1002' } } : { status: 401 }),
      'GET /api/v1/environment': () => new Response('Bad gateway', { status: 502 }),
      'GET /api/v1/recycle-bin': () => Promise.reject(new TypeError('Failed to fetch')),
    });

    expect((await fetch(`${ORIGIN}/api/v1/me`)).status).toBe(200);
    expect((await fetch(`${ORIGIN}/api/v1/me`)).status).toBe(401);
    expect(await (await fetch(`${ORIGIN}/api/v1/environment`)).text()).toBe('Bad gateway');
    await expect(fetch(`${ORIGIN}/api/v1/recycle-bin`)).rejects.toThrow('Failed to fetch');
  });

  it('remembers requests without a canned answer, so the test setup can fail the test', async () => {
    mockApi({});

    const response = await fetch(`${ORIGIN}/api/v1/sample-folders`);

    expect(response.status).toBe(501);
    // Taking them here keeps this test green; normally the setup's afterEach reports them.
    expect(takeUnmockedRequests()).toEqual(['GET /api/v1/sample-folders']);
    expect(takeUnmockedRequests()).toEqual([]);
  });

  it('answers the app frame’s background requests unless the test answers them itself', async () => {
    mockApi({ 'GET /api/v1/notifications': { body: { items: [], total: 3, unread: 3 } } });

    const summary = await fetch(`${ORIGIN}/api/v1/approvals/summary`);
    const notifications = await fetch(`${ORIGIN}/api/v1/notifications`);

    expect(await summary.json()).toEqual(FRAME_ANSWERS['GET /api/v1/approvals/summary']?.body);
    expect(await notifications.json()).toEqual({ items: [], total: 3, unread: 3 });
    expect(takeUnmockedRequests()).toEqual([]);
  });
});
