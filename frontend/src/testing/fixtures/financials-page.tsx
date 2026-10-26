import { screen } from '@testing-library/react';
import { Outlet } from 'react-router';
import { FinancialsPage, TransactionPanel } from '@/features/financials';
import { SessionProvider } from '@/session/SessionProvider';
import { mockApi } from '../mock-api';
import type { MockedApi } from '../mock-api';
import { renderRoutes } from '../render';
import { reply } from '../replies';
import type { Answer } from '../replies';
import { transactionPage } from './financials';
import { testUser } from './users';

/**
 * Renders the Financials page as the router does: the list at /financials and the details panel
 * as its child route. Other addresses show "Another page".
 */
export function renderFinancials(route = '/financials') {
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

/** The Financials page at `route`, its list answered by `list` (2026's 11 transactions). */
export function renderFinancialsPage(
  route = '/financials',
  list: Answer<'GET /api/v1/financials/transactions'> = { body: transactionPage() },
) {
  // A 403 makes the session provider reload the person's access (014 FR-020).
  const api = mockApi([
    reply('GET /api/v1/financials/transactions', list),
    reply('GET /api/v1/me', { body: testUser() }),
  ]);
  return { ...renderFinancials(route), api };
}

/** The query strings the page asked the list with, in order. */
export function listQueries(api: MockedApi): URLSearchParams[] {
  return api.requests
    .filter((request) => request.path === '/api/v1/financials/transactions')
    .map((request) => new URLSearchParams(request.search));
}

export const transactionRow = (name: string) => screen.getByRole('row', { name: new RegExp(name) });
