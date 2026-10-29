import { screen } from '@testing-library/react';
import type { HttpHandler } from 'msw';
import type { ActionPlan } from '@/features/plans/usePlanActions';
import { mockApi } from '../mock-api';
import { renderWithProviders } from '../render';
import { reply } from '../replies';
import { activityRecord } from './line-item-financials';
import { PlanActionsMenu } from './PlanActionsMenu';
import { testUser } from './users';

export const CHINA: ActionPlan = { id: 'P-1002', name: 'FY2026 Mobility China', status: 'draft' };
export const PUBLISHED: ActionPlan = {
  id: 'P-1003',
  name: 'FY2025 Lubricants Europe',
  status: 'published',
};

/** The ⋯ menu of `target` on the Plans list, with `answers` as the fake API. */
export function renderActions(target: ActionPlan, answers: HttpHandler[]) {
  const api = mockApi([
    ...answers,
    reply('GET /api/v1/financials/records/{recordId}', { body: activityRecord() }),
  ]);
  const result = renderWithProviders(<PlanActionsMenu target={target} />, {
    user: testUser(),
    route: '/plans',
    path: '/plans',
  });
  return { ...result, api };
}

/** Chooses `action` in the ⋯ menu of the plan called `name`. */
export async function choose(
  user: ReturnType<typeof renderActions>['user'],
  name: string,
  action: string,
) {
  await user.click(screen.getByRole('button', { name: `Actions for ${name}` }));
  await user.click(await screen.findByRole('menuitem', { name: action }));
}
