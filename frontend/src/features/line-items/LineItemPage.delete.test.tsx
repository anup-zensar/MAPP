import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { HISTORY_ANSWERS } from '@/testing/fixtures/activity-log';
import { details } from '@/testing/fixtures/line-items';
import { currentRates, deleteResult, formAnswers } from '@/testing/fixtures/line-item-forms';
import { lineItemRecord, transactionPage } from '@/testing/fixtures/line-item-financials';
import { lineItemSection } from '@/testing/fixtures/purchase-requests';
import { testUser } from '@/testing/fixtures/users';
import { mockApi } from '@/testing/mock-api';
import { findMessages, renderWithProviders } from '@/testing/render';
import { reply } from '@/testing/replies';
import { LineItemPage } from './LineItemPage';

function renderPage() {
  const api = mockApi([
    ...HISTORY_ANSWERS,
    ...formAnswers(),
    reply('GET /api/v1/exchange-rates/current', { body: currentRates() }),
    reply(
      'GET /api/v1/financials/records/{recordId}',
      { body: lineItemRecord() },
      { recordId: 'L-1001' },
    ),
    reply('GET /api/v1/financials/transactions', { body: transactionPage([]) }),
    reply('GET /api/v1/line-items/{lineItemId}', { body: details() }, { lineItemId: 'L-1001' }),
    reply('GET /api/v1/line-items/{lineItemId}/purchase-requests', {
      body: lineItemSection({ lineItemId: 'L-1001', purchaseRequests: [] }),
    }),
    reply(
      'DELETE /api/v1/line-items/{lineItemId}',
      { body: deleteResult() },
      { lineItemId: 'L-1001' },
    ),
  ]);
  return {
    api,
    ...renderWithProviders(<LineItemPage />, {
      route: '/line-items/L-1001',
      path: '/line-items/:lineItemId',
      user: testUser(),
    }),
  };
}

describe('038 FR-013 (L-3): delete from the line-item page', () => {
  it('never asks for the deleted line item again', async () => {
    const { user, api, router } = renderPage();
    await screen.findByRole('heading', { level: 1, name: 'Agency fee' });

    await user.click(screen.getByRole('button', { name: 'Actions for Agency fee' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));
    const dialog = await screen.findByRole('alertdialog', {
      name: "Delete line item 'Agency fee'?",
    });
    await user.click(within(dialog).getByRole('button', { name: 'Delete line item' }));

    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/plans/P-1001/activities/A-1001'),
    );
    await findMessages();
    const deleted = api.requests.findIndex((request) => request.method === 'DELETE');
    const after = api.requests.slice(deleted + 1).map((r) => `${r.method} ${r.path}`);
    expect(after.filter((request) => request.includes('L-1001'))).toEqual([]);
  });
});
