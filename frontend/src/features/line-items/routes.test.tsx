import { screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { HttpHandler } from 'msw';
import { HISTORY_ANSWERS } from '@/testing/fixtures/activity-log';
import { expectCurrentSection, renderApp } from '@/testing/app';
import {
  details,
  formAnswers,
  formOptions,
  lineItemRecord,
  transaction,
  transactionPage,
} from '@/testing/fixtures/line-items';
import { looseReply } from '@/testing/replies';

const FORM_OPTIONS: HttpHandler[] = [
  ...formAnswers(),
  looseReply(
    'GET /api/v1/activities/{activityId}/line-items/form-options',
    { body: formOptions() },
    { activityId: 'A-1001' },
  ),
];
const DETAILS: HttpHandler[] = [
  ...FORM_OPTIONS,
  ...HISTORY_ANSWERS,
  looseReply('GET /api/v1/line-items/{lineItemId}', { body: details() }, { lineItemId: 'L-1001' }),
  looseReply(
    'GET /api/v1/financials/records/{recordId}',
    { body: lineItemRecord() },
    { recordId: 'L-1001' },
  ),
  looseReply('GET /api/v1/financials/transactions', { body: transactionPage([transaction()]) }),
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
