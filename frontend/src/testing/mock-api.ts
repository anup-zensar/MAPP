import { http as msw, HttpResponse } from 'msw';
import type { HttpHandler } from 'msw';
import { setupServer } from 'msw/node';
import { ORIGIN, reply } from './replies';

/**
 * The fake MAPP API of component and page tests (research R19, 026 R9), on Mock Service Worker.
 * A request no handler answers gets a 501 and fails the test afterwards (testing/setup.ts).
 *
 *   const api = mockApi([reply('GET /api/v1/me', { body: priya })]);
 */

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
  /** The bodies sent to one "METHOD /path" (the path as sent, with its IDs), in order. */
  bodiesOf(key: string): unknown[];
}

/** A canned answer of the untyped form, until every test answers through `reply`. */
export interface MockAnswer {
  status?: number;
  body?: unknown;
}

export type MockHandler =
  MockAnswer | ((request: Request) => MockAnswer | Response | Promise<MockAnswer | Response>);

/**
 * The app frame's background requests (feature 008: the sidebar's Approvals count and the bell's
 * notifications, asked on every signed-in page), answered "nothing waiting" unless a test answers.
 */
export const FRAME_ANSWERS: HttpHandler[] = [
  reply('GET /api/v1/approvals/summary', {
    body: { waitingForMe: 0, allOpen: 0, approvedThisMonth: 0, declinedThisMonth: 0 },
  }),
  reply('GET /api/v1/notifications', { body: { items: [], total: 0, unread: 0 } }),
];

const TEST_HEADER = 'X-Test-Run';
let run = 0;
let recorded: RecordedRequest[] = [];
const unanswered: string[] = [];

/** A request a finished test left behind fails as a network error, unseen by the current test. */
const leftover = msw.all(`${ORIGIN}/*`, ({ request }) =>
  request.headers.get(TEST_HEADER) === String(run) ? undefined : HttpResponse.error(),
);

const recorder = msw.all(`${ORIGIN}/*`, async ({ request }) => {
  recorded.push(await describe(request));
});

const lastResort = msw.all(`${ORIGIN}/*`, ({ request }) => {
  const key = keyOf(request);
  unanswered.push(key);
  return HttpResponse.json({ message: `mockApi has no answer for ${key}` }, { status: 501 });
});

/** The fake network of every test; testing/setup.ts starts and stops it. */
export const server = setupServer(lastResort);

/**
 * Marks every request with the test that sends it. MSW answers a request a few ticks later, when
 * the next test may have begun; the mark keeps each answer and record with its own test.
 */
export function markRequests(): void {
  const intercepted = globalThis.fetch;
  globalThis.fetch = (input, init) => {
    const request = new Request(input, init);
    request.headers.set(TEST_HEADER, String(run));
    return intercepted(request);
  };
}

/** Answers this test's requests with `handlers` (a later one wins) and records every request. */
export function mockApi(handlers: HttpHandler[] | Record<string, MockHandler>): MockedApi {
  const list = Array.isArray(handlers) ? handlers : Object.entries(handlers).map(untyped);
  run += 1;
  recorded = [];
  const requests = recorded;
  server.resetHandlers();
  server.use(...FRAME_ANSWERS);
  server.use(...list.toReversed());
  server.use(leftover, recorder);
  return {
    requests,
    bodiesOf: (key) => requests.filter((r) => `${r.method} ${r.path}` === key).map((r) => r.body),
  };
}

/** Returns the requests that had no answer since the last call, and forgets them. */
export function takeUnmockedRequests(): string[] {
  return unanswered.splice(0);
}

function untyped([key, handler]: [string, MockHandler]): HttpHandler {
  const [method = '', path = ''] = key.split(' ');
  return msw.all(`${ORIGIN}${path}`, async ({ request }) => {
    if (request.method !== method) {
      return undefined;
    }
    try {
      const answer = typeof handler === 'function' ? await handler(request) : handler;
      return answer instanceof Response ? answer : jsonResponse(answer);
    } catch {
      return HttpResponse.error();
    }
  });
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

async function describe(request: Request): Promise<RecordedRequest> {
  const url = new URL(request.url);
  const text = await request.clone().text();
  return { method: request.method, path: url.pathname, search: url.search, body: parse(text) };
}

function keyOf(request: Request): string {
  return `${request.method} ${new URL(request.url).pathname}`;
}

function parse(text: string): unknown {
  if (text === '') {
    return undefined;
  }
  try {
    return JSON.parse(text);
  } catch {
    return text;
  }
}
