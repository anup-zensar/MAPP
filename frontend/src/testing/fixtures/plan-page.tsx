import { screen, within } from '@testing-library/react';
import type { HttpHandler } from 'msw';
import { PlanPage } from '@/features/plans';
import type { Plan } from '@/features/plans';
import { closestTo } from '../dom';
import { mockApi } from '../mock-api';
import { renderWithProviders } from '../render';
import { reply } from '../replies';
import type { Answer } from '../replies';
import { HISTORY_ANSWERS } from './activity-log';
import { planForm } from './attribute-forms';
import { planCampaigns, usd } from './campaigns';
import { financialsNotServed, plan } from './plans';
import { testUser } from './users';

/** The plan tree of a plan with nothing inside: no campaigns and no activities, for any plan. */
export const NOTHING_INSIDE: HttpHandler[] = [
  reply('GET /api/v1/plans/{planId}/campaigns', {
    body: planCampaigns([], {
      allocatedToCampaigns: usd('0.00'),
      planBudget: usd('1200000.00'),
      remainingPlanBudget: usd('1200000.00'),
    }),
  }),
  reply('GET /api/v1/plans/{planId}/activities', { body: { items: [], total: 0 } }),
];

/** The plan page of P-1001 at `route`: `answer` is GET /api/v1/plans/P-1001, `more` the rest. */
export function renderPlanPage(
  answer: Answer<'GET /api/v1/plans/{planId}'> = { body: plan() },
  more: HttpHandler[] = [],
  route = '/plans/P-1001',
) {
  const api = mockApi([
    ...HISTORY_ANSWERS,
    ...NOTHING_INSIDE,
    ...financialsNotServed('P-1001', 'P-1004'),
    reply('GET /api/v1/plans/{planId}', answer, { planId: 'P-1001' }),
    reply('GET /api/v1/shell-attribute-forms/{level}', { body: planForm() }, { level: 'plan' }),
    ...more,
  ]);
  const result = renderWithProviders(<PlanPage />, {
    user: testUser(),
    route,
    path: '/plans/:planId',
  });
  return { ...result, api };
}

export function withFigures(figures: Partial<Plan['figures']>, rest: Partial<Plan> = {}) {
  return { body: plan({ figures: { ...plan().figures, ...figures }, ...rest }) };
}

/** One of the summary cards, by its label. */
export function summaryCard(label: string): HTMLElement {
  const summary = screen.getByRole('region', { name: 'Summary' });
  return closestTo(within(summary).getByText(label), 'div');
}
