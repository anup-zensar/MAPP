import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/testing/mockApi';
import type { MockHandler } from '@/testing/mockApi';
import {
  FT_1003,
  FT_1004,
  FT_1005,
  manuallyLinked,
  STATION_SIGNAGE,
  transactionOf,
  transactionPage,
} from '@/testing/fixtures/financials';
import { Outlet } from 'react-router';
import { SessionProvider } from '@/session/SessionProvider';
import { renderRoutes } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { FinancialsPage } from './FinancialsPage';
import { TransactionPanel } from './TransactionPanel';

const LIST = 'GET /api/v1/financials/transactions';

function renderPanel(id: string, answer: MockHandler, more: Record<string, MockHandler> = {}) {
  const api = mockApi({
    [LIST]: { body: transactionPage() },
    [`GET /api/v1/financials/transactions/${id}`]: answer,
    ...more,
  });
  return { ...renderFinancials(`/financials/transactions/${id}?q=helix`), api };
}

/**
 * Renders the Financials page as the router does: the list at /financials and the details panel
 * as its child route. Other addresses show "Another page". For tests only.
 */
function renderFinancials(route = '/financials') {
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

describe('TransactionPanel (015 US3 scenarios 3–4, FR-022)', () => {
  it('shows every field, the linked path with a link per level and how it was matched', async () => {
    const { router, user } = renderPanel('FT-1003', { body: transactionOf(FT_1003) });

    const panel = await screen.findByRole('dialog', { name: 'PO 4500001003 / 10' });
    expect(within(panel).getByText('Purchase order · FT-1003')).toBeInTheDocument();
    const field = (label: string) =>
      within(panel).getByText(label, { selector: 'dt' }).nextElementSibling;
    expect(await within(panel).findByText('Document number')).toBeInTheDocument();
    expect(field('Document number')).toHaveTextContent('4500001003');
    expect(field('Line')).toHaveTextContent('10');
    expect(field('Posting date')).toHaveTextContent('08 Jan 2026');
    expect(field('Amount')).toHaveTextContent('9,800.00 EUR');
    expect(field('Amount in USD')).toHaveTextContent('$10,652.17');
    expect(field('Vendor')).toHaveTextContent('V-100103 · Outdoor Frames Ltd');
    expect(field('SWBS code')).toHaveTextContent('GB01-M26-0006');
    expect(field('HX key')).toHaveTextContent('HX1006');
    expect(field('Matched by')).toHaveTextContent('HX key HX1006');
    expect(field('From import')).toHaveTextContent(/^Demo data \(simulated GSAP\) on .+ by MAPP$/);

    const path = within(panel).getByRole('navigation', { name: 'Linked to' });
    expect(
      within(path)
        .getAllByRole('link')
        .map((link) => link.getAttribute('href')),
    ).toEqual([
      '/plans/P-1001',
      '/plans/P-1001/campaigns/C-1001',
      '/plans/P-1001/initiatives/I-1002',
      '/plans/P-1001/activities/A-1003',
      '/line-items/L-1006',
    ]);
    expect(within(path).getByRole('link', { name: 'Stand build (L-1006)' })).toBeInTheDocument();

    // A link made by GSAP data can't be changed here (US7 scenario 3).
    expect(
      within(panel).getByText('Linked by GSAP data (HX key HX1006) — correct it in GSAP.'),
    ).toBeInTheDocument();
    expect(within(panel).queryByRole('button', { name: 'Link to a line item' })).toBeNull();
    expect(within(panel).queryByRole('button', { name: 'Unlink' })).toBeNull();

    await user.click(within(panel).getAllByRole('button', { name: 'Close' }).at(-1)!);
    await waitFor(() => expect(router.state.location.pathname).toBe('/financials'));
    expect(router.state.location.search).toBe('?q=helix');
  });

  it('names the PO an invoice refers to', async () => {
    renderPanel('FT-1005', { body: transactionOf(FT_1005) });

    const panel = await screen.findByRole('dialog', { name: 'Invoice 5100002001' });
    const field = (label: string) =>
      within(panel).getByText(label, { selector: 'dt' }).nextElementSibling;
    expect(field('PO number / PO line')).toHaveTextContent('4500001001');
    expect(field('Amount in USD')).toHaveTextContent('$20,000.00');
    expect(field('Matched by')).toHaveTextContent('PO number 4500001001');
  });

  it('shows a not-linked transaction with its reason and Link to a line item (US3 scenario 4)', async () => {
    const { user } = renderPanel(
      'FT-1004',
      { body: transactionOf(FT_1004) },
      { 'GET /api/v1/line-items/L-1007': { body: STATION_SIGNAGE } },
    );

    const panel = await screen.findByRole('dialog', { name: 'PO 4500001004 / 10' });
    const field = (label: string) =>
      within(panel).getByText(label, { selector: 'dt' }).nextElementSibling;
    expect(field('Linked to')).toHaveTextContent('Not linked');
    expect(field('Reason')).toHaveTextContent(
      'No line item for HX key HX1099 — check the tracking key in GSAP.',
    );

    await user.click(within(panel).getByRole('button', { name: 'Link to a line item' }));
    expect(
      await screen.findByRole('dialog', { name: "Link 'PO 4500001004 / 10' to a line item" }),
    ).toBeInTheDocument();
  });

  it('unlinks a manual link after asking (US7 scenario 2)', async () => {
    const { api, user } = renderPanel(
      'FT-1004',
      { body: manuallyLinked() },
      {
        'POST /api/v1/financials/transactions/FT-1004/unlink': {
          body: transactionOf(FT_1004, { version: 3 }),
        },
      },
    );

    const panel = await screen.findByRole('dialog', { name: 'PO 4500001004 / 10' });
    const field = (label: string) =>
      within(panel).getByText(label, { selector: 'dt' }).nextElementSibling;
    expect(field('Matched by')).toHaveTextContent('Linked by Alex Morgan on 20 Oct 2026');
    expect(within(panel).queryByText(/correct it in GSAP/)).toBeNull();

    await user.click(within(panel).getByRole('button', { name: 'Unlink' }));
    const confirm = await screen.findByRole('alertdialog', {
      name: "Unlink 'PO 4500001004 / 10' from 'Station signage (L-1007)'?",
    });
    expect(within(confirm).getByRole('button', { name: 'Cancel' })).toHaveFocus();
    await user.click(within(confirm).getByRole('button', { name: 'Unlink' }));

    expect(
      (
        await screen.findAllByText("'PO 4500001004 / 10' unlinked from 'Station signage (L-1007)'")
      )[0],
    ).toBeInTheDocument();
    expect(api.bodiesOf('POST /api/v1/financials/transactions/FT-1004/unlink')).toEqual([
      { version: 2 },
    ]);
  });

  it('gives the focus to Link to a line item after Unlink, as its button is gone (WCAG 2.4.3)', async () => {
    let unlinked = false;
    const { user } = renderPanel(
      'FT-1004',
      () => ({ body: unlinked ? transactionOf(FT_1004, { version: 3 }) : manuallyLinked() }),
      {
        'POST /api/v1/financials/transactions/FT-1004/unlink': () => {
          unlinked = true;
          return { body: transactionOf(FT_1004, { version: 3 }) };
        },
      },
    );

    const panel = await screen.findByRole('dialog', { name: 'PO 4500001004 / 10' });
    await user.click(within(panel).getByRole('button', { name: 'Unlink' }));
    const confirm = await screen.findByRole('alertdialog');
    await user.click(within(confirm).getByRole('button', { name: 'Unlink' }));

    const link = await within(panel).findByRole('button', { name: 'Link to a line item' });
    await waitFor(() => expect(link).toHaveFocus());
    expect(within(panel).queryByRole('button', { name: 'Unlink' })).toBeNull();
  });

  it('gives the focus to Unlink after Link (US7 scenario 1)', async () => {
    let linked = false;
    const { user } = renderPanel(
      'FT-1004',
      () => ({ body: linked ? manuallyLinked() : transactionOf(FT_1004) }),
      {
        'GET /api/v1/line-items/L-1007': { body: STATION_SIGNAGE },
        'POST /api/v1/financials/transactions/FT-1004/link': () => {
          linked = true;
          return { body: manuallyLinked() };
        },
      },
    );

    const panel = await screen.findByRole('dialog', { name: 'PO 4500001004 / 10' });
    await user.click(within(panel).getByRole('button', { name: 'Link to a line item' }));
    const dialog = await screen.findByRole('dialog', {
      name: "Link 'PO 4500001004 / 10' to a line item",
    });
    await user.type(within(dialog).getByRole('textbox', { name: /Line item ID/ }), 'L-1007');
    await user.click(within(dialog).getByRole('button', { name: 'Link' }));

    const unlink = await within(panel).findByRole('button', { name: 'Unlink' });
    await waitFor(() => expect(unlink).toHaveFocus());
  });

  it('says when the transaction does not exist', async () => {
    renderPanel('FT-9999', {
      status: 404,
      body: { error: { code: 'not_found', message: "This GSAP transaction doesn't exist." } },
    });

    expect(await screen.findByText("This GSAP transaction doesn't exist.")).toBeInTheDocument();
  });
});
