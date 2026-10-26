import { screen } from '@testing-library/react';
import type { HttpHandler } from 'msw';
import { Outlet } from 'react-router';
import { CampaignPanel } from '@/features/plans';
import { SessionProvider } from '@/session/SessionProvider';
import { mockApi } from '../mock-api';
import { renderRoutes } from '../render';
import { reply } from '../replies';
import { HISTORY_ANSWERS } from './activity-log';
import { campaignForm } from './attribute-forms';
import { campaign, PEOPLE, planAnswer } from './campaigns';
import { financialsNotServed } from './plans';
import { testUser } from './users';

/** The details panel of Helix launch (C-1001) over its plan. */
export const DETAILS = '/plans/P-1001/campaigns/C-1001';

/** The campaign panels at `route`; `answers` add to or replace what the panel asks. */
export function renderPanel(route: string, answers: HttpHandler[] = []) {
  const api = mockApi([
    ...HISTORY_ANSWERS,
    ...financialsNotServed('C-1001'),
    reply('GET /api/v1/campaigns/{campaignId}', { body: campaign() }, { campaignId: 'C-1001' }),
    reply('GET /api/v1/plans/{planId}', { body: planAnswer() }, { planId: 'P-1001' }),
    reply(
      'GET /api/v1/shell-attribute-forms/{level}',
      { body: campaignForm() },
      { level: 'campaign' },
    ),
    reply('GET /api/v1/people', { body: PEOPLE }),
    ...answers,
  ]);
  const rendered = renderRoutes(
    [
      {
        element: (
          <SessionProvider>
            <Outlet />
          </SessionProvider>
        ),
        children: [
          {
            path: '/plans/:planId',
            element: <Outlet />,
            children: [
              { path: 'campaigns/:campaignId', element: <CampaignPanel mode="details" /> },
              { path: 'campaigns/:campaignId/edit', element: <CampaignPanel mode="edit" /> },
              { path: '*', element: <p>Another panel</p> },
              { index: true, element: <p>Plan page</p> },
            ],
          },
          { path: '*', element: <p>Another page</p> },
        ],
      },
    ],
    { route, user: testUser() },
  );
  return { api, ...rendered };
}

export async function findDetails(name = 'Helix launch') {
  return screen.findByRole('dialog', { name });
}

/** The edit form once it is shown (the loading panel is another one). */
export async function findForm() {
  await screen.findByLabelText('Campaign name');
  await screen.findByText(/Alex Morgan/);
  return screen.getByRole('dialog', { name: 'Edit campaign' });
}
