import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { closestTo } from '@/testing/dom';
import { findMessages } from '@/testing/render';
import { DIGITAL, OLD, PAGE_2026 } from '@/testing/fixtures/finance-budget-items';
import { deleted } from '@/testing/fixtures/settings';
import { chooseAction, renderFinanceBudgetItemsPage } from '@/testing/fixtures/settings-pages';
import { reply } from '@/testing/replies';

function budgetCard(name: string) {
  return screen.getByRole('region', { name });
}

describe('FinanceBudgetItemsPage: the list (FR-014, FR-015)', () => {
  it('shows the current year, one card per budget with its active items (US5 scenario 3)', async () => {
    renderFinanceBudgetItemsPage();

    expect(
      screen.getByRole('heading', { level: 1, name: 'Finance budget items' }),
    ).toBeInTheDocument();
    expect(document.title).toBe('Finance budget items · MAPP');
    expect(
      screen.getByText(
        'The finance budget items line items link to. A simple list until the Finance module arrives.',
      ),
    ).toBeInTheDocument();

    const lubricants = await screen.findByRole('region', { name: '2026 Lubricants Budget' });
    expect(lubricants).toHaveTextContent('3 items · Default finance budget of 2 plans');
    const digital = closestTo(
      within(lubricants).getByRole('rowheader', { name: 'Europe – Digital' }),
      'tr',
    );
    expect(digital).toHaveTextContent('LUB-EU-DIG');
    expect(digital).toHaveTextContent('Linked to 6 line items');
    expect(within(lubricants).queryByText('Old item')).not.toBeInTheDocument();
    expect(budgetCard('2026 Mobility Budget')).toHaveTextContent('1 item · Not used yet');
    const years = screen.getByRole('group', { name: 'Budget year' });
    expect(within(years).getByRole('button', { name: '2026' })).toHaveAttribute(
      'aria-pressed',
      'true',
    );
    expect(screen.getByText('3 of 4 finance budget items')).toBeInTheDocument();
  });

  it('switches the budget year, keeping it in the address', async () => {
    const { user, router, api } = renderFinanceBudgetItemsPage();

    await screen.findByRole('region', { name: '2026 Lubricants Budget' });
    await user.click(
      within(screen.getByRole('group', { name: 'Budget year' })).getByRole('button', {
        name: '2027',
      }),
    );

    expect(
      await screen.findByText('No finance budget items for 2027 yet — create the first one'),
    ).toBeInTheDocument();
    expect(router.state.location.search).toBe('?year=2027');
    expect(api.requests.at(-1)?.search).toBe('?year=2027');
    expect(screen.getAllByRole('button', { name: 'New finance budget item' })).toHaveLength(2);
  });

  it('searches items, budgets and codes', async () => {
    const { user } = renderFinanceBudgetItemsPage();

    await screen.findByRole('region', { name: '2026 Lubricants Budget' });
    await user.type(
      screen.getByRole('searchbox', { name: 'Search finance budget items' }),
      'mob-flt',
    );

    expect(
      screen.queryByRole('region', { name: '2026 Lubricants Budget' }),
    ).not.toBeInTheDocument();
    expect(within(budgetCard('2026 Mobility Budget')).getByText('Fleet')).toBeInTheDocument();
    expect(screen.getByText('1 of 4 finance budget items')).toBeInTheDocument();
  });

  it('says when nothing matches the filters, and clears them', async () => {
    const { user } = renderFinanceBudgetItemsPage([], '/settings/finance-budget-items?q=zzz');

    expect(await screen.findByText('Nothing matches these filters')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Clear filters' }));

    expect(within(budgetCard('2026 Lubricants Budget')).getByText('Old item')).toBeInTheDocument();
  });

  it('says when the budgets cannot be loaded, and tries again', async () => {
    let calls = 0;
    const { user } = renderFinanceBudgetItemsPage([
      reply('GET /api/v1/finance-budgets', () => {
        calls += 1;
        return calls === 1
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
          : { body: PAGE_2026 };
      }),
    ]);

    expect(await screen.findByRole('alert')).toHaveTextContent('Something went wrong.');
    await user.click(screen.getByRole('button', { name: 'Try again' }));

    expect(
      await screen.findByRole('region', { name: '2026 Lubricants Budget' }),
    ).toBeInTheDocument();
  });
});

describe('FinanceBudgetItemsPage: details (US5 scenario 4)', () => {
  it('shows the line items linked to it and the plans using its budget', async () => {
    const { user, router } = renderFinanceBudgetItemsPage();

    await user.click(await screen.findByRole('button', { name: 'Europe – Digital' }));

    const panel = await screen.findByRole('dialog', { name: 'Europe – Digital' });
    expect(router.state.location.search).toBe('?id=FBI-1001');
    expect(panel).toHaveTextContent('Budget year2026');
    expect(panel).toHaveTextContent('2026 Lubricants Budget');
    expect(panel).toHaveTextContent('Default finance budget of 2 plans');
    expect(within(panel).getByRole('link', { name: 'FY2026 Lubricants Europe' })).toHaveAttribute(
      'href',
      '/plans/P-1001',
    );
    expect(panel).toHaveTextContent('CodeLUB-EU-DIG');
    expect(panel).toHaveTextContent('Linked to 6 line items');
    expect(panel).toHaveTextContent(
      'Created by Priya Raman on 12 Mar 2026, last changed by Daniel Okoro on 14 Mar 2026',
    );
  });

  it('says when the item of the address no longer exists', async () => {
    renderFinanceBudgetItemsPage([], '/settings/finance-budget-items?id=FBI-9999');

    const panel = await screen.findByRole('dialog', { name: 'Finance budget item' });
    expect(panel).toHaveTextContent(
      'This finance budget item no longer exists — someone may have deleted it.',
    );
    expect(
      within(panel).getByRole('button', { name: 'Back to Finance budget items' }),
    ).toBeVisible();
  });
});

describe('FinanceBudgetItemsPage: status and delete (US5 scenario 5, FR-004)', () => {
  it('moves an unused item to the recycle bin and offers Undo', async () => {
    const { user } = renderFinanceBudgetItemsPage([
      reply(
        'DELETE /api/v1/finance-budget-items/{itemId}',
        {
          body: deleted('finance budget item', 'FBI-1002', 'Asia – Retail'),
        },
        { itemId: 'FBI-1002' },
      ),
    ]);

    await chooseAction('Asia – Retail', 'Delete', user);
    const dialog = await screen.findByRole('alertdialog', {
      name: "Delete finance budget item 'Asia – Retail'?",
    });
    await user.click(within(dialog).getByRole('button', { name: 'Delete finance budget item' }));

    const messages = await findMessages();
    expect(messages).toHaveTextContent(
      "Finance budget item 'Asia – Retail' moved to the recycle bin",
    );
    expect(within(messages).getByRole('button', { name: 'Undo' })).toBeInTheDocument();
  });

  it('offers Set inactive instead of deleting an item in use', async () => {
    const { user, api } = renderFinanceBudgetItemsPage([
      reply(
        'POST /api/v1/finance-budget-items/{itemId}/status',
        { body: { ...DIGITAL, active: false } },
        { itemId: 'FBI-1001' },
      ),
    ]);

    await chooseAction('Europe – Digital', 'Delete', user);
    const dialog = await screen.findByRole('alertdialog', { name: "'Europe – Digital' is in use" });
    expect(dialog).toHaveTextContent(
      "Linked to 6 line items — set it inactive instead. Line items can't link to an inactive item; those linked keep it.",
    );
    await user.click(within(dialog).getByRole('button', { name: 'Set inactive' }));

    expect(await findMessages()).toHaveTextContent(
      "Finance budget item 'Europe – Digital' set inactive",
    );
    expect(api.bodiesOf('POST /api/v1/finance-budget-items/FBI-1001/status')).toEqual([
      { active: false, version: 2 },
    ]);
  });

  it('reactivates an inactive item', async () => {
    const { user, api } = renderFinanceBudgetItemsPage(
      [
        reply(
          'POST /api/v1/finance-budget-items/{itemId}/status',
          { body: { ...OLD, active: true } },
          { itemId: 'FBI-1003' },
        ),
      ],
      '/settings/finance-budget-items?status=inactive',
    );

    await chooseAction('Old item', 'Reactivate', user);

    expect(await findMessages()).toHaveTextContent("Finance budget item 'Old item' reactivated");
    expect(api.bodiesOf('POST /api/v1/finance-budget-items/FBI-1003/status')).toEqual([
      { active: true, version: 1 },
    ]);
  });
});
