import { screen } from '@testing-library/react';
import { Outlet } from 'react-router';
import { FinancialsPage, TransactionPanel } from '@/features/financials';
import { SessionProvider } from '@/session/SessionProvider';
import { mockApi, server } from '../mock-api';
import type { MockedApi } from '../mock-api';
import { renderRoutes } from '../render';
import { reply } from '../replies';
import type { Answer } from '../replies';
import { transactionPage } from './financials';
import { runsPage } from './swbs-runs';
import { testUser } from './users';

// The Financials page as the router renders it; the simulated GSAP panel (030) is answered here.
export function renderFinancials(route = '/financials') {
  server.use(reply('GET /api/v1/simulated-gsap/swbs-runs', { body: runsPage() }));
  return renderRoutes(
    [
      {
        element: (
          <SessionProvider>
            <Outlet />
          </SessionProvider>
        ),
        children: [
          {
            path: '/financials',
            element: <FinancialsPage />,
            children: [{ path: 'transactions/:transactionId', element: <TransactionPanel /> }],
          },
          { path: '*', element: <p>Another page</p> },
        ],
      },
    ],
    { route, user: testUser() },
  );
}

export function renderFinancialsPage(
  route = '/financials',
  list: Answer<'GET /api/v1/financials/transactions'> = { body: transactionPage() },
) {
  const api = mockApi([
    reply('GET /api/v1/financials/transactions', list),
    reply('GET /api/v1/me', { body: testUser() }),
  ]);
  return { ...renderFinancials(route), api };
}

export function listQueries(api: MockedApi): URLSearchParams[] {
  return api.requests
    .filter((request) => request.path === '/api/v1/financials/transactions')
    .map((request) => new URLSearchParams(request.search));
}

export const transactionRow = (name: string) => screen.getByRole('row', { name: new RegExp(name) });
