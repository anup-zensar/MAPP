/** The fake MAPP API of component and page tests, on Mock Service Worker (research R19, 026 R9). */
import { http as msw, HttpResponse } from 'msw';
import type { HttpHandler } from 'msw';
import { setupServer } from 'msw/node';
import { NO_BREAKDOWN } from './fixtures/all-work-breakdown';
import { configuration } from './fixtures/purchase-request-pages';
import { ORIGIN, reply } from './replies';

export interface RecordedRequest {
  method: string;
  path: string;
  search: string;
  body: unknown;
}

export interface MockedApi {
  requests: RecordedRequest[];
  bodiesOf(key: string): unknown[];
}

export const FRAME_ANSWERS: HttpHandler[] = [
  reply('GET /api/v1/approvals/summary', {
    body: { waitingForMe: 0, allOpen: 0, approvedThisMonth: 0, declinedThisMonth: 0 },
  }),
  reply('GET /api/v1/notifications', { body: { items: [], total: 0, unread: 0 } }),
  reply('GET /api/v1/overspend-configuration', {
    body: {
      defaultMode: 'warn',
      approvalOverPlannedCost: 'warn',
      version: 1,
      updatedAt: '2026-03-12T09:00:00Z',
      updatedBy: { id: 'system', name: 'SP&A One' },
      canChange: { available: true, reason: '' },
    },
  }),
  reply('GET /api/v1/field-help', { body: { items: [], total: 0, shellAttributesWithoutHelp: 0 } }),
  reply('GET /api/v1/purchase-request-configuration', { body: configuration() }),
  ...NO_BREAKDOWN,
];

const TEST_HEADER = 'X-Test-Run';
let run = 0;
let recorded: RecordedRequest[] = [];
const unanswered: string[] = [];

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

export const server = setupServer(lastResort);

export function markRequests(): void {
  const intercepted = globalThis.fetch;
  globalThis.fetch = (input, init) => {
    const request = new Request(input, init);
    request.headers.set(TEST_HEADER, String(run));
    return intercepted(request);
  };
}

export function mockApi(handlers: HttpHandler[]): MockedApi {
  run += 1;
  recorded = [];
  const requests = recorded;
  server.resetHandlers();
  server.use(...FRAME_ANSWERS);
  server.use(...handlers.toReversed());
  server.use(leftover, recorder);
  return {
    requests,
    bodiesOf: (key) => requests.filter((r) => `${r.method} ${r.path}` === key).map((r) => r.body),
  };
}

export function takeUnmockedRequests(): string[] {
  return unanswered.splice(0);
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
