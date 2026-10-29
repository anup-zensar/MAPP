import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { HISTORY_ANSWERS } from '@/testing/fixtures/activity-log';
import { lineItemRecord, transactionPage } from '@/testing/fixtures/line-item-financials';
import { currentRates, formAnswers } from '@/testing/fixtures/line-item-forms';
import { CREATE_ROUTE } from '@/testing/fixtures/line-item-pages';
import { details } from '@/testing/fixtures/line-items';
import { lineItemSection } from '@/testing/fixtures/purchase-requests';
import { VIEW_ONLY_NOTE, VIEWER, VIEWER_LINE_ITEM_ACCESS } from '@/testing/fixtures/viewer';
import { expectNoAxeViolations } from '@/testing/axe';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { apiError, reply } from '@/testing/replies';
import { LineItemCreatePage } from './LineItemCreatePage';
import { LineItemPage } from './LineItemPage';

const REFUSED =
  "Changing activities and line items isn't part of your role (Viewer). Ask an administrator if you need it.";

describe('014 FR-011: the line-item page follows the reader’s access', () => {
  it('shows a Viewer one View only note, no Edit, and only View history in More', async () => {
    mockApi([
      ...HISTORY_ANSWERS,
      ...formAnswers(),
      reply('GET /api/v1/exchange-rates/current', { body: currentRates() }),
      reply('GET /api/v1/financials/records/{recordId}', { body: lineItemRecord() }),
      reply('GET /api/v1/financials/transactions', { body: transactionPage([]) }),
      reply('GET /api/v1/line-items/{lineItemId}', {
        body: { ...details(), access: VIEWER_LINE_ITEM_ACCESS },
      }),
      reply('GET /api/v1/line-items/{lineItemId}/purchase-requests', {
        body: lineItemSection({ lineItemId: 'L-1001', purchaseRequests: [] }),
      }),
    ]);
    const { user } = renderWithProviders(<LineItemPage />, {
      route: '/line-items/L-1001',
      path: '/line-items/:lineItemId',
      user: VIEWER,
    });

    expect(await screen.findByRole('note')).toHaveTextContent(VIEW_ONLY_NOTE);
    expect(screen.queryByRole('link', { name: 'Edit' })).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: /^Actions for/ }));
    expect(screen.getAllByRole('menuitem').map((item) => item.textContent)).toEqual([
      'View history',
    ]);
    await user.keyboard('{Escape}');
    await expectNoAxeViolations();
  });

  it('shows the no-access page when the API refuses the new line-item form (FR-008)', async () => {
    mockApi([
      ...formAnswers(),
      reply('GET /api/v1/activities/{activityId}/line-items/form-options', {
        status: 403,
        body: apiError('forbidden', REFUSED),
      }),
      reply('GET /api/v1/me', { body: VIEWER }),
    ]);
    renderWithProviders(<LineItemCreatePage />, {
      route: CREATE_ROUTE,
      path: '/activities/:activityId/line-items/new',
      user: VIEWER,
    });

    expect(await screen.findByRole('heading', { level: 1, name: 'No access' })).toBeVisible();
    expect(screen.getByText(REFUSED)).toBeVisible();
  });
});
