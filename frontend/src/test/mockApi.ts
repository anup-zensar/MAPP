import { vi } from 'vitest';

/**
 * A fake NorthStar API for component and page tests (research R19).
 *
 * mockApi() replaces the browser's fetch with canned answers, one per "METHOD /path":
 *
 *   const api = mockApi({
 *     'GET /api/me': { body: priya },
 *     'PATCH /api/me/preferences': { status: 400, body: validationError },
 *   });
 *
 * A request without a canned answer is remembered, and the test setup fails the test afterwards,
 * so a missing answer never looks like a server that cannot be reached. The test setup also puts
 * the real fetch back after every test.
 */

/** A canned answer: a status (200 when left out) and a JSON body (no body when left out). */
export interface MockAnswer {
  status?: number;
  body?: unknown;
}

/**
 * What an endpoint answers: a canned answer, or a function for anything special — answers that
 * change from call to call, a Response that is not JSON, a request that never finishes or a
 * network failure (a rejected promise).
 */
export type MockHandler =
  MockAnswer | ((request: Request) => MockAnswer | Response | Promise<MockAnswer | Response>);

/** One request the code under test made. */
export interface RecordedRequest {
  method: string;
  path: string;
  /** The query string, for example "?limit=20" ("" when there is none). */
  search: string;
  /** The JSON body, parsed (undefined when there is no body). */
  body: unknown;
}

/** The fake API of one test. */
export interface MockedApi {
  /** Every request, in the order they were made. */
  requests: RecordedRequest[];
  /** The bodies sent to one "METHOD /path", in order. */
  bodiesOf(key: string): unknown[];
}

/**
 * The app frame's background requests (feature 008: the sidebar's Approvals count and the bell's
 * notifications, asked on every signed-in page). mockApi answers them with "nothing waiting"
 * unless the test gives its own answer, so page tests of any feature need not mock them.
 */
export const FRAME_ANSWERS: Record<string, MockAnswer> = {
  'GET /api/approvals/summary': {
    body: { waitingForMe: 0, allOpen: 0, approvedThisMonth: 0, declinedThisMonth: 0 },
  },
  'GET /api/notifications': { body: { items: [], total: 0, unread: 0 } },
};

/** Requests that had no canned answer, as "METHOD /path"; read by the test setup. */
const unmockedRequests: string[] = [];

/** Replaces fetch with the given canned answers for the rest of the test. */
export function mockApi(handlers: Record<string, MockHandler>): MockedApi {
  const requests: RecordedRequest[] = [];

  vi.stubGlobal('fetch', async (input: RequestInfo | URL, init?: RequestInit) => {
    const request = new Request(input, init);
    const url = new URL(request.url);
    const key = `${request.method} ${url.pathname}`;
    requests.push({
      method: request.method,
      path: url.pathname,
      search: url.search,
      body: parseBody(await request.clone().text()),
    });

    const handler = handlers[key] ?? FRAME_ANSWERS[key];
    if (!handler) {
      unmockedRequests.push(key);
      return jsonResponse({ status: 501, body: { message: `mockApi has no answer for ${key}` } });
    }
    const answer = typeof handler === 'function' ? await handler(request) : handler;
    return answer instanceof Response ? answer : jsonResponse(answer);
  });

  return {
    requests,
    bodiesOf: (key) => requests.filter((r) => `${r.method} ${r.path}` === key).map((r) => r.body),
  };
}

/** Returns the requests that had no canned answer since the last call, and forgets them. */
export function takeUnmockedRequests(): string[] {
  return unmockedRequests.splice(0);
}

function jsonResponse({ status = 200, body }: MockAnswer): Response {
  if (body === undefined) {
    return new Response(null, { status });
  }
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

function parseBody(text: string): unknown {
  if (text === '') {
    return undefined;
  }
  try {
    return JSON.parse(text);
  } catch {
    return text;
  }
}
