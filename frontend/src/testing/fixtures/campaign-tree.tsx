import type { HttpHandler } from 'msw';
import type { InitialEntry } from 'react-router';
import { Outlet } from 'react-router';
import { NewCampaignPanel } from '@/features/plans';
import { SessionProvider } from '@/session/SessionProvider';
import { mockApi } from '../mock-api';
import { renderRoutes } from '../render';
import { reply } from '../replies';
import { CampaignTreeStandIn } from './CampaignTreeStandIn';
import { activityRecord } from './line-item-financials';
import { testUser } from './users';

/** The campaign rows of P-1001 in a stand-in plan page, at `route`, with New campaign below it. */
export function renderTree(answers: HttpHandler[], route: InitialEntry = '/plans/P-1001') {
  // Feature 006 adds the plan's activities under the initiatives: none here.
  const api = mockApi([
    reply(
      'GET /api/v1/plans/{planId}/activities',
      { body: { items: [], total: 0 } },
      { planId: 'P-1001' },
    ),
    ...answers,
    reply('GET /api/v1/financials/records/{recordId}', { body: activityRecord() }),
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
            element: <CampaignTreeStandIn />,
            children: [
              { path: 'campaigns/new', element: <NewCampaignPanel /> },
              { path: '*', element: <p>A panel</p> },
            ],
          },
        ],
      },
    ],
    { route, user: testUser() },
  );
  return { api, ...rendered };
}
