import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { HttpHandler } from 'msw';
import { HISTORY_ANSWERS } from '@/testing/fixtures/activity-log';
import { financialsNotServed } from '@/testing/fixtures/plans';
import { closestTo } from '@/testing/dom';
import { mockApi } from '@/testing/mock-api';
import { findMessages, renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import {
  approvedActivity,
  details,
  filmProduction,
  lineItem,
  usd,
} from '@/testing/fixtures/line-items';
import {
  currentRates,
  deleteResult,
  formAnswers,
  notFound,
} from '@/testing/fixtures/line-item-forms';
import {
  lineItemRecord,
  transaction,
  transactionPage,
} from '@/testing/fixtures/line-item-financials';
import { lineItemSection } from '@/testing/fixtures/purchase-requests';
import { reply } from '@/testing/replies';
import { LineItemPage } from './LineItemPage';

const DUPLICATE = 'POST /api/v1/line-items/L-1001/duplicate';

function renderPage(answers: HttpHandler[] = []) {
  const api = mockApi([
    ...HISTORY_ANSWERS,
    ...formAnswers(),
    reply('GET /api/v1/exchange-rates/current', { body: currentRates() }),
    reply(
      'GET /api/v1/financials/records/{recordId}',
      { body: lineItemRecord() },
      { recordId: 'L-1001' },
    ),
    reply('GET /api/v1/financials/transactions', {
      body: transactionPage([
        transaction({
          id: 'FT-1003',
          type: 'invoice',
          name: 'Invoice 5100002003',
          postingDate: '2026-04-02',
          amount: usd('12000.00'),
          amountDisplay: usd('12000.00'),
        }),
        transaction(),
      ]),
    }),
    reply('GET /api/v1/line-items/{lineItemId}', { body: details() }, { lineItemId: 'L-1001' }),
    reply('GET /api/v1/line-items/{lineItemId}/purchase-requests', {
      body: lineItemSection({ lineItemId: 'L-1001', purchaseRequests: [] }),
    }),
    ...answers,
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

async function openMore(user: ReturnType<typeof renderPage>['user'], item: string) {
  await user.click(screen.getByRole('button', { name: 'Actions for Agency fee' }));
  await user.click(await screen.findByRole('menuitem', { name: item }));
}

describe('LineItemPage (US4, FR-026)', () => {
  it('shows where the line item lives, its chips and facts (scenario 1)', async () => {
    renderPage();

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Agency fee' }),
    ).toBeInTheDocument();
    expect(document.title).toBe('Agency fee · SP&A One');
    const breadcrumb = screen.getByLabelText('Breadcrumb');
    expect(within(breadcrumb).getByRole('link', { name: 'Launch film' })).toHaveAttribute(
      'href',
      '/plans/P-1001/activities/A-1001',
    );
    expect(within(breadcrumb).getByText('Agency fee')).toHaveAttribute('aria-current', 'page');
    expect(screen.getByText('LINE ITEM')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '3 fields needed' })).toBeInTheDocument();
    expect(screen.getAllByText('Waiting for activity approval')).toHaveLength(2);
    expect(screen.getByText('L-1001')).toBeInTheDocument();
    expect(screen.getAllByText('Priya Raman').length).toBeGreaterThan(0);
    expect(screen.getAllByText('50,000.00 USD').length).toBeGreaterThan(0);
    expect(
      screen.getByText(
        'Created by Priya Raman on 12 Mar 2026, last changed by Daniel Okoro on 14 Mar 2026',
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Edit' })).toHaveAttribute(
      'href',
      '/line-items/L-1001/edit',
    );
    expect(screen.getByRole('button', { name: 'Actions for Agency fee' })).toBeInTheDocument();
    const march = closestTo(screen.getByRole('cell', { name: 'Mar 2026' }), 'tr');
    expect(within(march).getAllByRole('cell')[1]).toHaveTextContent('16,666.66 USD');
    expect(screen.getAllByText('—').length).toBeGreaterThan(0);
    // Details and Finance coding: two cards with the same header, no heading repeated (LI-03).
    expect(screen.getByRole('region', { name: 'Details' })).toBeInTheDocument();
    const coding = await screen.findByRole('region', { name: 'Finance coding' });
    expect(within(coding).getAllByRole('heading')).toHaveLength(1);
    expect(within(coding).getByText('Request SWBS code from GSAP')).toBeInTheDocument();
    // GSAP's money, open (015 FR-017): figures and every transaction.
    const financials = await screen.findByRole('region', { name: 'Financials' });
    expect(within(financials).getAllByText('30,000.00 USD').length).toBeGreaterThan(0);
    expect(
      await within(financials).findByRole('table', { name: /transactions, newest first/ }),
    ).toBeInTheDocument();
    expect(within(financials).getByText('Invoice 5100002003')).toBeInTheDocument();
    expect(screen.queryByText('From finance (coming later)')).not.toBeInTheDocument();
  });

  it('shows a planned cost in another currency with its USD amount', async () => {
    renderPage([
      reply(
        'GET /api/v1/line-items/{lineItemId}',
        { body: details({ lineItem: { ...filmProduction(), id: 'L-1001' } }) },
        { lineItemId: 'L-1001' },
      ),
    ]);

    // The summary card: the amount in its currency, the USD amount and the rate used under it.
    const summary = await screen.findByRole('region', { name: 'Summary' });
    expect(within(summary).getByText('46,000.00 EUR')).toBeInTheDocument();
    expect(within(summary).getByText('= 50,000.00 USD at 0.92 EUR per USD')).toBeInTheDocument();
  });

  it('offers Duplicate, View history and Delete under More', async () => {
    const { user, router } = renderPage();
    await screen.findByRole('heading', { level: 1, name: 'Agency fee' });

    await user.click(screen.getByRole('button', { name: 'Actions for Agency fee' }));
    const items = (await screen.findAllByRole('menuitem')).map((item) => item.textContent);
    expect(items).toEqual(['Duplicate', 'View history', 'Delete']);
    await user.click(screen.getByRole('menuitem', { name: 'View history' }));

    // The page's History section gets the focus (feature 011, FR-011).
    await waitFor(() =>
      expect(screen.getByRole('heading', { level: 2, name: 'History' })).toHaveFocus(),
    );
    expect(router.state.location.pathname).toBe('/line-items/L-1001');
  });

  it('ends with the line item’s History, without an include option (011 FR-011, FR-012)', async () => {
    const { api } = renderPage();
    await screen.findByRole('heading', { level: 1, name: 'Agency fee' });

    const history = screen.getByRole('region', { name: 'History' });
    expect(within(history).getByText('Every change to this line item, newest first')).toBeVisible();
    expect(within(history).queryByRole('checkbox')).not.toBeInTheDocument();
    expect(await within(history).findByText('No changes recorded yet.')).toBeVisible();
    const asked = api.requests.find((request) => request.path === '/api/v1/activity-log');
    expect(new URLSearchParams(asked?.search).get('recordId')).toBe('L-1001');
  });

  it('duplicates and opens the copy (scenario 4, FR-029)', async () => {
    const { user, api, router } = renderPage([
      reply(
        'POST /api/v1/line-items/{lineItemId}/duplicate',
        { status: 201, body: lineItem({ id: 'L-2002', name: 'Agency fee (copy)' }) },
        { lineItemId: 'L-1001' },
      ),
      reply(
        'GET /api/v1/line-items/{lineItemId}',
        {
          body: details({ lineItem: lineItem({ id: 'L-2002', name: 'Agency fee (copy)' }) }),
        },
        { lineItemId: 'L-2002' },
      ),
      ...financialsNotServed('L-2002'),
    ]);
    await screen.findByRole('heading', { level: 1, name: 'Agency fee' });

    await openMore(user, 'Duplicate');

    await waitFor(() => expect(router.state.location.pathname).toBe('/line-items/L-2002'));
    expect(api.bodiesOf(DUPLICATE)).toEqual([{}]);
    expect(await screen.findByText("Line item 'Agency fee (copy)' created")).toBeInTheDocument();
  });

  it('034 FR-001: duplicates in an approved activity at once, without asking (D-322)', async () => {
    const { user, api, router } = renderPage([
      reply(
        'GET /api/v1/line-items/{lineItemId}',
        { body: details({ activity: approvedActivity() }) },
        { lineItemId: 'L-1001' },
      ),
      reply(
        'POST /api/v1/line-items/{lineItemId}/duplicate',
        { status: 201, body: lineItem({ id: 'L-2002', name: 'Agency fee (copy)' }) },
        { lineItemId: 'L-1001' },
      ),
      reply(
        'GET /api/v1/line-items/{lineItemId}',
        {
          body: details({ lineItem: lineItem({ id: 'L-2002', name: 'Agency fee (copy)' }) }),
        },
        { lineItemId: 'L-2002' },
      ),
      ...financialsNotServed('L-2002'),
    ]);
    await screen.findByRole('heading', { level: 1, name: 'Agency fee' });

    await openMore(user, 'Duplicate');

    await waitFor(() => expect(router.state.location.pathname).toBe('/line-items/L-2002'));
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    expect(api.bodiesOf(DUPLICATE)).toEqual([{}]);
  });

  it('deletes after the dialog and offers Undo (scenario 5, FR-030)', async () => {
    const { user, api, router } = renderPage([
      reply(
        'DELETE /api/v1/line-items/{lineItemId}',
        { body: deleteResult() },
        { lineItemId: 'L-1001' },
      ),
      reply(
        'POST /api/v1/recycle-bin/{entryId}/undo',
        {
          body: {
            recordType: 'lineItem',
            recordTypeLabel: 'line item',
            recordId: 'L-1001',
            recordName: 'Agency fee',
            details: '',
          },
        },
        { entryId: 'BIN-1001' },
      ),
    ]);
    await screen.findByRole('heading', { level: 1, name: 'Agency fee' });

    await openMore(user, 'Delete');
    const dialog = await screen.findByRole('alertdialog', {
      name: "Delete line item 'Agency fee'?",
    });
    expect(dialog).toHaveTextContent(
      "The activity's unassigned planned cost goes up by 50,000.00 USD.",
    );
    expect(dialog).toHaveTextContent('It moves to the recycle bin; you can restore it.');
    // The GSAP transactions linked to it stay (015 FR-029), from the Financials section's query.
    expect(dialog).toHaveTextContent(
      '2 GSAP transactions are linked to it. They stay in SP&A One and leave the figures until it is restored.',
    );
    expect(dialog).not.toHaveTextContent('This activity will need approval again.');
    expect(within(dialog).getByRole('button', { name: 'Cancel' })).toHaveFocus();

    await user.click(within(dialog).getByRole('button', { name: 'Delete line item' }));

    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/plans/P-1001/activities/A-1001'),
    );
    expect(router.state.location.hash).toBe('#line-items');
    expect(api.requests.find((r) => r.method === 'DELETE')?.search).toBe('');
    expect(
      await screen.findByText("Line item 'Agency fee' deleted"),
    ).toBeInTheDocument();
    const messages = await findMessages();
    await user.click(within(messages).getByRole('button', { name: 'Undo' }));
    expect(await screen.findByText("Line item 'Agency fee' restored")).toBeInTheDocument();
  });

  it('034 FR-001: deletes from an approved activity without any word of approval (D-322)', async () => {
    const { user, api } = renderPage([
      reply(
        'GET /api/v1/line-items/{lineItemId}',
        { body: details({ activity: approvedActivity() }) },
        { lineItemId: 'L-1001' },
      ),
      reply(
        'DELETE /api/v1/line-items/{lineItemId}',
        { body: deleteResult() },
        { lineItemId: 'L-1001' },
      ),
    ]);
    await screen.findByRole('heading', { level: 1, name: 'Agency fee' });

    await openMore(user, 'Delete');
    const dialog = await screen.findByRole('alertdialog', {
      name: "Delete line item 'Agency fee'?",
    });
    expect(dialog).not.toHaveTextContent(/approval/i);
    await user.click(within(dialog).getByRole('button', { name: 'Delete line item' }));

    await waitFor(() => expect(api.requests.find((r) => r.method === 'DELETE')?.search).toBe(''));
  });

  it.each([
    ['This line item no longer exists — someone may have deleted it.', 'Go to Plans', '/plans'],
    [
      'This activity was deleted — restore it from the recycle bin to keep working on its line items.',
      'Go to the recycle bin',
      '/recycle-bin',
    ],
  ])('says when it is gone: %s (D-154)', async (message, link, href) => {
    renderPage([
      reply('GET /api/v1/line-items/{lineItemId}', notFound(message), {
        lineItemId: 'L-1001',
      }),
    ]);

    expect(await screen.findByText(message)).toBeInTheDocument();
    expect(screen.getByRole('link', { name: link })).toHaveAttribute('href', href);
  });
});
