import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/testing/mockApi';
import type { MockHandler } from '@/testing/mockApi';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { LineItemsSection } from './LineItemsSection';
import {
  activity,
  activityRecord,
  deleteResult,
  launchFilmSection,
  lineItem,
  lineItemRecord,
  usd,
} from '@/testing/fixtures/line-items';

const SECTION = 'GET /api/v1/activities/A-1001/line-items';
const FINANCIALS = 'GET /api/v1/financials/records/A-1001';

function renderSection(
  answer: MockHandler,
  state?: unknown,
  financials: MockHandler = { body: activityRecord() },
) {
  mockApi({ [SECTION]: answer, [FINANCIALS]: financials });
  return renderWithProviders(<LineItemsSection activityId="A-1001" />, {
    route: { pathname: '/plans/P-1001/activities/A-1001', state },
    user: testUser(),
  });
}

describe('LineItemsSection (US1)', () => {
  it('lists each cost with its amounts, spread and chips (scenario 1, FR-001)', async () => {
    renderSection({ body: launchFilmSection() });

    const table = await screen.findByRole('table', { name: 'Line items' });
    const rows = within(table).getAllByRole('row');
    expect(rows).toHaveLength(3);
    const film = rows[2];
    expect(within(film).getByRole('link', { name: 'Film production' })).toHaveAttribute(
      'href',
      '/line-items/L-1002',
    );
    expect(film).toHaveTextContent('L-1002');
    expect(film).toHaveTextContent('EUR');
    expect(film).toHaveTextContent('46,000.00 EUR');
    expect(film).toHaveTextContent('50,000.00 USD');
    expect(film).toHaveTextContent('Evenly');
    expect(film).toHaveTextContent('All filled in');
    expect(film).toHaveTextContent('Not requested');
    expect(rows[1]).toHaveTextContent('3 fields needed');
    expect(rows[1]).toHaveTextContent('Waiting for activity approval');
    expect(
      within(table).getByRole('columnheader', { name: 'Planned cost (USD)' }),
    ).toBeInTheDocument();
  });

  it('shows Committed and Invoiced from GSAP on every row, a commitment above planned cost in red words (015 FR-016)', async () => {
    renderSection({ body: launchFilmSection() });

    const table = await screen.findByRole('table', { name: 'Line items' });
    const heads = within(table)
      .getAllByRole('columnheader')
      .map((head) => head.textContent);
    expect(heads.slice(1, 5)).toEqual([
      'Planned cost',
      'Planned cost (USD)',
      'Committed (USD)',
      'Invoiced (USD)',
    ]);
    const [, fee, film] = within(table).getAllByRole('row');
    await waitFor(() =>
      expect(within(fee).getAllByRole('cell')[3]).toHaveTextContent('30,000.00 USD'),
    );
    expect(within(fee).getAllByRole('cell')[4]).toHaveTextContent('12,000.00 USD');
    expect(within(film).getAllByRole('cell')[3]).toHaveTextContent('52,000.00 USD');
    expect(
      within(within(film).getAllByRole('cell')[3]).getByRole('img', {
        name: 'Committed above planned cost',
      }),
    ).toBeInTheDocument();
  });

  it('shows "—" in the GSAP columns when the figures cannot be read', async () => {
    renderSection({ body: launchFilmSection() }, undefined, {
      status: 409,
      body: { error: { code: 'not_allowed', message: 'Not allowed', fields: [] } },
    });

    const table = await screen.findByRole('table', { name: 'Line items' });
    const [, fee] = within(table).getAllByRole('row');
    await waitFor(() => expect(within(fee).getAllByRole('cell')[3]).toHaveTextContent('—'));
  });

  it('shows the unassigned planned cost with its meter (scenario 1, FR-002)', async () => {
    renderSection({ body: launchFilmSection() });

    expect(
      await screen.findByText('Unassigned planned cost (Sys_Gen_Balance)'),
    ).toBeInTheDocument();
    expect(screen.getByText('20,000.00 USD')).toBeInTheDocument();
    expect(screen.getByRole('progressbar', { name: 'Assigned to line items' })).toHaveAttribute(
      'aria-valuenow',
      '83',
    );
    expect(screen.getByText('100,000.00 of 120,000.00 USD assigned')).toBeInTheDocument();
    expect(screen.getByText('20,000.00 USD not yet assigned to a line item')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'About Unassigned planned cost' }),
    ).toBeInTheDocument();
  });

  it('shows the grid by month with totals (scenario 2, FR-003)', async () => {
    renderSection({ body: launchFilmSection() });

    const grid = await screen.findByRole('table', { name: 'Line items spread over time, in USD' });
    const headers = within(grid)
      .getAllByRole('columnheader')
      .map((header) => header.textContent);
    expect(headers).toEqual(['Line item', 'Mar 2026', 'Apr 2026', 'May 2026', 'Total']);
    expect(within(grid).getByRole('row', { name: /^Total/ })).toHaveTextContent(
      'Total33,333.3233,333.3233,333.36100,000.00',
    );
  });

  it('says in red when line items exceed the planned cost (scenario 3, D-016)', async () => {
    const section = launchFilmSection({
      balance: {
        plannedCost: usd('120000.00'),
        assigned: usd('120500.00'),
        balance: usd('-500.00'),
        state: 'exceeded',
        roundingHint: false,
      },
    });
    renderSection({ body: section });

    expect(
      await screen.findByText("Line items exceed the activity's planned cost by 500.00 USD"),
    ).toBeInTheDocument();
    expect(screen.getByText('-500.00 USD')).toBeInTheDocument();
  });

  it('shows the empty state and the whole planned cost as unassigned (scenario 4, FR-004)', async () => {
    const section = launchFilmSection({
      items: [],
      balance: {
        plannedCost: usd('120000.00'),
        assigned: usd('0.00'),
        balance: usd('120000.00'),
        state: 'left',
        roundingHint: false,
      },
      grid: { columns: [], rows: [], totals: [], total: usd('0.00') },
    });
    renderSection({ body: section });

    expect(
      await screen.findByText(
        "No line items yet — split this activity's planned cost into the costs you will pay",
      ),
    ).toBeInTheDocument();
    for (const link of screen.getAllByRole('link', { name: 'Add line item' })) {
      expect(link).toHaveAttribute('href', '/activities/A-1001/line-items/new');
    }
    expect(screen.getByText('120,000.00 USD not yet assigned to a line item')).toBeInTheDocument();
    expect(screen.queryByRole('table')).not.toBeInTheDocument();
  });

  it('shows the SWBS request status of each line item (scenario 5, FR-024)', async () => {
    const section = launchFilmSection({
      activity: activity({ status: 'approved' }),
      items: [
        lineItem({ id: 'L-1', name: 'One', swbsStatus: 'readyToSend' }),
        lineItem({ id: 'L-2', name: 'Two', swbsStatus: 'waitingForApproval' }),
        lineItem({ id: 'L-3', name: 'Three', requestSwbs: false, swbsStatus: 'notRequested' }),
      ],
    });
    renderSection({ body: section });

    expect(await screen.findByText('Ready to send to GSAP')).toBeInTheDocument();
    expect(screen.getByText('Waiting for activity approval')).toBeInTheDocument();
    expect(screen.getByText('Not requested')).toBeInTheDocument();
  });

  it('says when everything is assigned, when there is no planned cost and when cents are left', async () => {
    const allAssigned = launchFilmSection({
      balance: {
        plannedCost: usd('100000.03'),
        assigned: usd('100000.00'),
        balance: usd('0.03'),
        state: 'left',
        roundingHint: true,
      },
    });
    renderSection({ body: allAssigned });
    expect(
      await screen.findByText(
        "If a currency rate leaves a few cents unassigned, adjust the activity's planned cost instead.",
      ),
    ).toBeInTheDocument();
  });

  it.each([
    [
      {
        plannedCost: usd('100000.00'),
        assigned: usd('100000.00'),
        balance: usd('0.00'),
        state: 'allAssigned' as const,
      },
      'All of the planned cost is assigned',
    ],
    [
      {
        plannedCost: null,
        assigned: usd('100000.00'),
        balance: null,
        state: 'noPlannedCost' as const,
      },
      'This activity has no planned cost yet — add one to see how much is left to assign.',
    ],
  ])('words the balance state %#', async (balance, text) => {
    renderSection({ body: launchFilmSection({ balance: { ...balance, roundingHint: false } }) });
    expect(await screen.findByText(text)).toBeInTheDocument();
  });

  it('labels the USD columns with the display currency and shows the amounts the API converted (FR-005, D-213)', async () => {
    const section = launchFilmSection();
    section.items = section.items.map((item) => ({
      ...item,
      plannedCostDisplay: { amount: '46000.00', currency: 'EUR' },
    }));
    section.balance = {
      ...section.balance,
      assigned: { amount: '92000.00', currency: 'EUR' },
      plannedCost: { amount: '110400.00', currency: 'EUR' },
      balance: { amount: '18400.00', currency: 'EUR' },
    };
    section.grid = { ...section.grid, total: { amount: '92000.00', currency: 'EUR' } };
    renderSection({ body: section });

    expect(
      await screen.findByRole('columnheader', { name: 'Planned cost (EUR)' }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('table', { name: 'Line items spread over time, in EUR' }),
    ).toBeInTheDocument();
    expect(screen.getByText('92,000.00 of 110,400.00 EUR assigned')).toBeInTheDocument();
    // Agency fee: 50,000.00 USD typed, shown as 46,000.00 EUR in the display-currency column.
    const fee = screen.getByRole('link', { name: 'Agency fee' }).closest('tr') as HTMLElement;
    expect(within(fee).getAllByRole('cell')[2]).toHaveTextContent('46,000.00 EUR');
  });

  it('highlights the row just created (FR-010)', async () => {
    renderSection({ body: launchFilmSection() }, { highlightLineItemId: 'L-1002' });

    const link = await screen.findByRole('link', { name: 'Film production' });
    expect(link.closest('tr')?.className).toMatch(/highlight/);
    expect(screen.getByRole('link', { name: 'Agency fee' }).closest('tr')?.className).not.toMatch(
      /highlight/,
    );
  });

  it('offers View, Edit, Duplicate and Delete in each row', async () => {
    const { user, router } = renderSection({ body: launchFilmSection() });

    await user.click(await screen.findByRole('button', { name: 'Actions for Agency fee' }));
    const items = (await screen.findAllByRole('menuitem')).map((item) => item.textContent);
    expect(items).toEqual(['View', 'Edit', 'Duplicate', 'Delete']);
    await user.click(screen.getByRole('menuitem', { name: 'Edit' }));
    await waitFor(() => expect(router.state.location.pathname).toBe('/line-items/L-1001/edit'));
  });

  it('deletes from the row and offers Undo, staying on the activity (FR-030)', async () => {
    mockApi({
      [SECTION]: { body: launchFilmSection() },
      [FINANCIALS]: { body: activityRecord() },
      'GET /api/v1/financials/records/L-1001': { body: lineItemRecord() },
      'DELETE /api/v1/line-items/L-1001': { body: deleteResult() },
    });
    const { user, router } = renderWithProviders(<LineItemsSection activityId="A-1001" />, {
      route: '/plans/P-1001/activities/A-1001',
      user: testUser(),
    });

    await user.click(await screen.findByRole('button', { name: 'Actions for Agency fee' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));
    const dialog = await screen.findByRole('alertdialog', {
      name: "Delete line item 'Agency fee'?",
    });
    // The GSAP transactions linked to it stay (015 FR-029).
    expect(
      await within(dialog).findByText(/^2 GSAP transactions are linked to it\./),
    ).toBeVisible();
    await user.click(within(dialog).getByRole('button', { name: 'Delete line item' }));

    expect(
      await screen.findByText("Line item 'Agency fee' moved to the recycle bin"),
    ).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/plans/P-1001/activities/A-1001');
  });

  it('opens a needed field from the completeness chip (FR-028)', async () => {
    const { user, router } = renderSection({ body: launchFilmSection() });

    await user.click(await screen.findByRole('button', { name: '3 fields needed' }));
    await user.click(await screen.findByRole('button', { name: 'Finance budget item' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/line-items/L-1001/edit'));
    expect(router.state.location.search).toBe('?field=financeBudgetItemId');
  });

  it('shows a skeleton while loading and Try again after a failure', async () => {
    let calls = 0;
    const { user } = renderSection(() =>
      calls++ === 0
        ? {
            status: 500,
            body: {
              error: {
                code: 'internal_error',
                message: 'Something went wrong. Please try again.',
                fields: [],
              },
            },
          }
        : { body: launchFilmSection() },
    );

    expect(screen.getByText('Loading…')).toBeInTheDocument();
    await user.click(await screen.findByRole('button', { name: 'Try again' }));
    expect(await screen.findByRole('table', { name: 'Line items' })).toBeInTheDocument();
  });
});
