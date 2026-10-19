import { describe, expect, it } from 'vitest';
import { mockApi, takeUnmockedRequests } from './mockApi';

const ORIGIN = window.location.origin;

describe('mockApi', () => {
  it('answers a request with the canned JSON for its "METHOD /path"', async () => {
    mockApi({ 'GET /api/health': { body: { status: 'ok' } } });

    const response = await fetch(`${ORIGIN}/api/health`);

    expect(response.status).toBe(200);
    expect(response.headers.get('Content-Type')).toBe('application/json');
    expect(await response.json()).toEqual({ status: 'ok' });
  });

  it('uses the given status and sends no body when there is none', async () => {
    mockApi({ 'POST /api/auth/sign-out': { status: 204 } });

    const response = await fetch(`${ORIGIN}/api/auth/sign-out`, { method: 'POST' });

    expect(response.status).toBe(204);
    expect(await response.text()).toBe('');
  });

  it('matches on the path only, so the query string does not matter', async () => {
    const api = mockApi({ 'GET /api/activity-log': { body: { items: [], total: 0 } } });

    const response = await fetch(`${ORIGIN}/api/activity-log?limit=20`);

    expect(response.status).toBe(200);
    expect(api.requests[0]).toMatchObject({ method: 'GET', path: '/api/activity-log' });
    expect(api.requests[0].search).toBe('?limit=20');
  });

  it('records every request with its JSON body', async () => {
    const api = mockApi({ 'PATCH /api/me/preferences': { body: { theme: 'dark' } } });

    await fetch(`${ORIGIN}/api/me/preferences`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ theme: 'dark' }),
    });

    expect(api.requests).toEqual([
      { method: 'PATCH', path: '/api/me/preferences', search: '', body: { theme: 'dark' } },
    ]);
    expect(api.bodiesOf('PATCH /api/me/preferences')).toEqual([{ theme: 'dark' }]);
  });

  it('lets a function answer, for responses in order, slow answers or failures', async () => {
    let calls = 0;
    mockApi({
      'GET /api/me': () => (++calls === 1 ? { body: { id: 'U-1002' } } : { status: 401 }),
      'GET /api/health': () => new Response('Bad gateway', { status: 502 }),
      'GET /api/recycle-bin': () => Promise.reject(new TypeError('Failed to fetch')),
    });

    expect((await fetch(`${ORIGIN}/api/me`)).status).toBe(200);
    expect((await fetch(`${ORIGIN}/api/me`)).status).toBe(401);
    expect(await (await fetch(`${ORIGIN}/api/health`)).text()).toBe('Bad gateway');
    await expect(fetch(`${ORIGIN}/api/recycle-bin`)).rejects.toThrow('Failed to fetch');
  });

  it('remembers requests without a canned answer, so the test setup can fail the test', async () => {
    mockApi({});

    const response = await fetch(`${ORIGIN}/api/sample-folders`);

    expect(response.status).toBe(501);
    // Taking them here keeps this test green; normally the setup's afterEach reports them.
    expect(takeUnmockedRequests()).toEqual(['GET /api/sample-folders']);
    expect(takeUnmockedRequests()).toEqual([]);
  });
});
