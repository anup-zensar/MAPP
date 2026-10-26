import { screen } from '@testing-library/react';
import type { HttpHandler } from 'msw';
import { ApprovalWorkflowsPage } from '@/features/approval-workflows';
import { mockApi } from '../mock-api';
import { renderWithProviders } from '../render';
import { reply } from '../replies';
import { CHINA_WORKFLOW, PEOPLE, workflowOf } from './approval-workflows';
import { testUser } from './users';

const ITEMS = [CHINA_WORKFLOW, workflowOf()];

/** GET /api/v1/approval-workflows with these workflows (China BC and Lubricants EU by default). */
export const workflowList = (items = ITEMS) => ({ body: { items, total: items.length } });

export function renderWorkflowsPage(
  route = '/settings/approval-workflows',
  answers: HttpHandler[] = [],
) {
  const api = mockApi([
    reply('GET /api/v1/approval-workflows', workflowList()),
    reply('GET /api/v1/people', PEOPLE),
    ...answers,
  ]);
  const rendered = renderWithProviders(<ApprovalWorkflowsPage />, {
    route,
    path: '/settings/approval-workflows/:workflowId?',
    user: testUser(),
  });
  return { ...rendered, api };
}

export const workflowRow = (name: string) => screen.getByRole('row', { name: new RegExp(name) });
