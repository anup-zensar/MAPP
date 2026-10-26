import { screen } from '@testing-library/react';
import type { HttpHandler } from 'msw';
import { GlobalBudgetPage } from '@/features/global-budgets';
import { mockApi } from '../mock-api';
import { renderWithProviders } from '../render';
import { reply } from '../replies';
import { globalBudget } from './global-budgets';
import { testUser } from './users';

/** The page of GB-1001 "FY2026 Global marketing budget", at /budgets/GB-1001. */
export function renderBudgetPage(answers: HttpHandler[] = []) {
  const api = mockApi([
    reply(
      'GET /api/v1/global-budgets/{globalBudgetId}',
      { body: globalBudget() },
      { globalBudgetId: 'GB-1001' },
    ),
    ...answers,
  ]);
  return {
    ...renderWithProviders(<GlobalBudgetPage />, {
      route: '/budgets/GB-1001',
      path: '/budgets/:globalBudgetId',
      user: testUser(),
    }),
    api,
  };
}

export const budgetRow = (name: string) => screen.getByRole('row', { name: new RegExp(name) });
