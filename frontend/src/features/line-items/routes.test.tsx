import { screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { HttpHandler } from 'msw';
import { HISTORY_ANSWERS } from '@/testing/fixtures/activity-log';
import { expectCurrentSection, renderApp } from '@/testing/app';
import { details } from '@/testing/fixtures/line-items';
import { formAnswers, formOptions } from '@/testing/fixtures/line-item-forms';
import {
  lineItemRecord,
  transaction,
  transactionPage,
} from '@/testing/fixtures/line-item-financials';
import { lineItemSection } from '@/testing/fixtures/purchase-requests';
import { reply } from '@/testing/replies';

const FORM_OPTIONS: HttpHandler[] = [
  ...formAnswers(),
  reply(
    'GET /api/v1/activities/{activityId}/line-items/form-options',
    { body: formOptions() },
    { activityId: 'A-1001' },
  ),
];
const DETAILS: HttpHandler[] = [
  ...FORM_OPTIONS,
  ...HISTORY_ANSWERS,
  reply('GET /api/v1/line-items/{lineItemId}', { body: details() }, { lineItemId: 'L-1001' }),
  reply(
    'GET /api/v1/financials/records/{recordId}',
    { body: lineItemRecord() },
    { recordId: 'L-1001' },
  ),
  reply('GET /api/v1/financials/transactions', { body: transactionPage([transaction()]) }),
  reply('GET /api/v1/line-items/{lineItemId}/purchase-requests', {
    body: lineItemSection({ lineItemId: 'L-1001', purchaseRequests: [] }),
  }),
];

// Feature 007's addresses (build guide §5), with the requests each page makes.
describe('Line item routes (feature 007)', () => {
  it.each<[string, string, string, HttpHandler[]]>([
    ['/activities/A-1001/line-items/new', 'New line item', 'New line item', FORM_OPTIONS],
    ['/line-items/L-1001', 'Agency fee', 'Agency fee', DETAILS],
    ['/line-items/L-1001/edit', 'Edit line item', 'Edit Agency fee', DETAILS],
  ])('opens %s: "%s", with Plans current in the sidebar', async (path, heading, tab, answers) => {
    renderApp(path, answers);

    expect(await screen.findByRole('heading', { level: 1, name: heading })).toBeInTheDocument();
    await waitFor(() => expect(document.title).toBe(`${tab} · MAPP`));
    expectCurrentSection('Plans');
  });
});
