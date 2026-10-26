import { screen, within } from '@testing-library/react';
import type { HttpHandler } from 'msw';
import { AllWorkPage } from '@/features/all-work';
import type { AllWorkResult, AllWorkRow } from '@/features/all-work';
import { mockApi } from '../mock-api';
import type { MockedApi } from '../mock-api';
import { renderWithProviders } from '../render';
import { http, reply } from '../replies';
import { gridResult, PAGE_ANSWERS } from './all-work';
import { row } from './all-work-rows';
import { testUser } from './users';

/** All work at `route` for Priya, on its first visit; `answers` win over the seed story's. */
export function renderAllWorkPage(answers: HttpHandler[] = [], route = '/all-work') {
  const api = mockApi([
    ...PAGE_ANSWERS,
    reply('GET /api/v1/all-work', { body: gridResult() }),
    ...answers,
  ]);
  const rendered = renderWithProviders(<AllWorkPage />, {
    user: testUser(),
    route,
    path: '/all-work',
  });
  return { ...rendered, api };
}

/** PUT /api/v1/all-work/preferences, answering what was sent, as the API does. */
export const PREFERENCES_SAVED = http.put(
  '/api/v1/all-work/preferences',
  async ({ request, response }) => response(200).json(await request.json()),
);

/** The query strings sent to GET /api/v1/all-work, in order. */
export function viewQueries(api: MockedApi): URLSearchParams[] {
  return api.requests
    .filter((request) => request.method === 'GET' && request.path === '/api/v1/all-work')
    .map((request) => new URLSearchParams(request.search));
}

export function rowNames(): string[] {
  const table = screen.getByRole('table', { name: 'All work grid' });
  return within(table)
    .getAllByRole('rowheader')
    .map((header) => within(header).queryByRole('link')?.textContent ?? header.textContent ?? '');
}

/** A plan with one campaign and initiative holding `count` activities. */
export function manyActivities(count: number): AllWorkResult {
  const activities: AllWorkRow[] = Array.from({ length: count }, (_, index) =>
    row({
      key: `A-${2000 + index}`,
      level: 'activity',
      name: `Activity ${index + 1}`,
      depth: 3,
      parentKey: 'I-1001',
      planId: 'P-1001',
      status: 'created',
      owner: 'Priya Raman',
    }),
  );
  return gridResult({
    rows: [...gridResult().rows.slice(0, 3), ...activities],
    totals: { ...gridResult().totals, activities: count },
  });
}
