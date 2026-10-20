import { Outlet } from 'react-router';
import { SessionProvider } from '@/app/SessionProvider';
import { renderRoutes } from '@/test/render';
import { testUser } from '@/test/users';
import { FinancialsPage } from './FinancialsPage';
import { TransactionPanel } from './TransactionPanel';

/**
 * Renders the Financials page as the router does: the list at /financials and the details panel
 * as its child route. Other addresses show "Another page". For tests only.
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
