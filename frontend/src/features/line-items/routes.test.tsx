import { screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { HISTORY_ANSWERS } from '@/features/activity-log/testData';
import { expectCurrentSection, renderApp } from '@/test/app';
import type { MockHandler } from '@/test/mockApi';
import {
  details,
  formAnswers,
  formOptions,
  lineItemRecord,
  transaction,
  transactionPage,
} from './testData';

const FORM_OPTIONS: Record<string, MockHandler> = {
  ...formAnswers(),
  'GET /api/activities/A-1001/line-items/form-options': { body: formOptions() },
};
const DETAILS: Record<string, MockHandler> = {
  ...FORM_OPTIONS,
  ...HISTORY_ANSWERS,
  'GET /api/line-items/L-1001': { body: details() },
  'GET /api/financials/records/L-1001': { body: lineItemRecord() },
  'GET /api/financials/transactions': { body: transactionPage([transaction()]) },
};

// Feature 007's addresses (build guide §5), with the requests each page makes.
describe('Line item routes (feature 007)', () => {
  it.each<[string, string, string, Record<string, MockHandler>]>([
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
