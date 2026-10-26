import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { components } from '@/api/generated/schema';
import { mockApi } from '@/testing/mockApi';
import type { MockHandler } from '@/testing/mockApi';
import { findMessages, renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import {
  deleted,
  MADE,
  NOT_USED,
  USED_BY_TWO_PLANS,
  validationFailed,
  versionConflict,
} from '@/testing/fixtures/settings';
import { FinanceBudgetItemsPage } from './FinanceBudgetItemsPage';

type FinanceBudget = components['schemas']['FinanceBudget'];
type FinanceBudgetItem = components['schemas']['FinanceBudgetItem'];

const YEAR_LOCK =
  "The year can't change once the item is saved — create a new item for another year.";

function item(overrides: Partial<FinanceBudgetItem> & { id: string; name: string }) {
  return {
    budgetId: 'FB-1001',
    budgetName: '2026 Lubricants Budget',
    year: 2026,
    code: '',
    active: true,
    usage: NOT_USED,
    locks: [{ field: 'year', reason: YEAR_LOCK }],
    version: 1,
    ...MADE,
    ...overrides,
  } satisfies FinanceBudgetItem;
}

const DIGITAL = item({
  id: 'FBI-1001',
  name: 'Europe – Digital',
  code: 'LUB-EU-DIG',
  usage: {
    inUse: true,
    summary: 'Linked to 6 line items',
    parts: [{ kind: 'lineItem', text: 'Linked to 6 line items', count: 6, inRecycleBin: 0 }],
    records: [],
  },
  version: 2,
});
const ASIA = item({ id: 'FBI-1002', name: 'Asia – Retail' });
const OLD = item({ id: 'FBI-1003', name: 'Old item', active: false });
const FLEET = item({
  id: 'FBI-1004',
  name: 'Fleet',
  code: 'MOB-FLT',
  budgetId: 'FB-1002',
  budgetName: '2026 Mobility Budget',
});

/** "2026 Lubricants Budget", the default budget of two plans. */
const LUBRICANTS_BUDGET: FinanceBudget = {
  id: 'FB-1001',
  year: 2026,
  name: '2026 Lubricants Budget',
  items: [ASIA, DIGITAL, OLD],
  activeItemCount: 2,
  usage: {
    ...USED_BY_TWO_PLANS,
    summary: 'Default finance budget of 2 plans',
    parts: [{ kind: 'plan', text: 'Default finance budget of 2 plans', count: 2, inRecycleBin: 0 }],
  },
  version: 4,
  ...MADE,
};

const MOBILITY_BUDGET: FinanceBudget = {
  ...LUBRICANTS_BUDGET,
  id: 'FB-1002',
  name: '2026 Mobility Budget',
  items: [FLEET],
  activeItemCount: 1,
  usage: NOT_USED,
  version: 1,
};

const YEARS = [2025, 2026, 2027, 2028, 2029, 2030, 2031];

const PAGE_2026 = {
  items: [LUBRICANTS_BUDGET, MOBILITY_BUDGET],
  total: 2,
  year: 2026,
  years: YEARS,
};
const PAGE_2027 = { items: [], total: 0, year: 2027, years: YEARS };

/** The finance budgets of the year asked for (this year, 2026, when none is given). */
const BUDGETS: MockHandler = (request) =>
  new URL(request.url).searchParams.get('year') === '2027'
    ? { body: PAGE_2027 }
    : { body: PAGE_2026 };

function renderItems(
  answers: Record<string, MockHandler> = {},
  route = '/settings/finance-budget-items',
) {
  const api = mockApi({ 'GET /api/v1/finance-budgets': BUDGETS, ...answers });
  return {
    ...renderWithProviders(<FinanceBudgetItemsPage />, {
      route,
      user: testUser(),
      path: '/settings/finance-budget-items',
    }),
    api,
  };
}

function budgetCard(name: string) {
  return screen.getByRole('region', { name });
}

async function chooseAction(
  name: string,
  action: string,
  user: ReturnType<typeof renderItems>['user'],
) {
  await user.click(await screen.findByRole('button', { name: `Actions for ${name}` }));
  await user.click(await screen.findByRole('menuitem', { name: action }));
}

describe('FinanceBudgetItemsPage: the list (FR-014, FR-015)', () => {
  it('shows the current year, one card per budget with its active items (US5 scenario 3)', async () => {
    renderItems();

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
    const digital = within(lubricants)
      .getByRole('rowheader', { name: 'Europe – Digital' })
      .closest('tr') as HTMLElement;
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
    const { user, router, api } = renderItems();

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
    const { user } = renderItems();

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
    const { user } = renderItems({}, '/settings/finance-budget-items?q=zzz');

    expect(await screen.findByText('Nothing matches these filters')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Clear filters' }));

    expect(within(budgetCard('2026 Lubricants Budget')).getByText('Old item')).toBeInTheDocument();
  });

  it('says when the budgets cannot be loaded, and tries again', async () => {
    let calls = 0;
    const { user } = renderItems({
      'GET /api/v1/finance-budgets': () => {
        calls += 1;
        return calls === 1
          ? {
              status: 500,
              body: {
                error: {
                  code: 'internal_error',
                  message: 'Something went wrong. Please try again.',
                },
              },
            }
          : { body: PAGE_2026 };
      },
    });

    expect(await screen.findByRole('alert')).toHaveTextContent('Something went wrong.');
    await user.click(screen.getByRole('button', { name: 'Try again' }));

    expect(
      await screen.findByRole('region', { name: '2026 Lubricants Budget' }),
    ).toBeInTheDocument();
  });
});

describe('FinanceBudgetItemsPage: details (US5 scenario 4)', () => {
  it('shows the line items linked to it and the plans using its budget', async () => {
    const { user, router } = renderItems();

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
    renderItems({}, '/settings/finance-budget-items?id=FBI-9999');

    const panel = await screen.findByRole('dialog', { name: 'Finance budget item' });
    expect(panel).toHaveTextContent(
      'This finance budget item no longer exists — someone may have deleted it.',
    );
    expect(
      within(panel).getByRole('button', { name: 'Back to Finance budget items' }),
    ).toBeVisible();
  });
});

describe('FinanceBudgetItemsPage: create and edit (US5 scenarios 3 and 4, FR-014)', () => {
  it('creates "Europe – Retail" in an existing budget, offering its name while typing', async () => {
    const created = item({ id: 'FBI-1005', name: 'Europe – Retail', code: 'LUB-EU-RET' });
    const { user, api } = renderItems({
      'POST /api/v1/finance-budget-items': { status: 201, body: created },
    });

    await screen.findByRole('region', { name: '2026 Lubricants Budget' });
    await user.click(screen.getByRole('button', { name: 'New finance budget item' }));
    const form = await screen.findByRole('dialog', { name: 'New finance budget item' });
    expect(within(form).getByRole('combobox', { name: /^Budget year/ })).toHaveValue('2026');
    const budgetName = within(form).getByRole('combobox', { name: /^Budget name/ });
    expect(budgetName).toHaveAccessibleDescription(
      expect.stringContaining('Items with the same year and budget name form one finance budget.'),
    );
    await waitFor(() =>
      expect(
        [...document.querySelectorAll('datalist option')].map((option) =>
          option.getAttribute('value'),
        ),
      ).toEqual(['2026 Lubricants Budget', '2026 Mobility Budget']),
    );
    await user.type(budgetName, '2026 Lubricants Budget');
    await user.type(within(form).getByRole('textbox', { name: /^Item name/ }), 'Europe – Retail');
    await user.type(within(form).getByRole('textbox', { name: /^Code/ }), 'LUB-EU-RET');
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    expect(await findMessages()).toHaveTextContent("Finance budget item 'Europe – Retail' created");
    expect(api.bodiesOf('POST /api/v1/finance-budget-items')).toEqual([
      {
        year: 2026,
        budgetName: '2026 Lubricants Budget',
        name: 'Europe – Retail',
        code: 'LUB-EU-RET',
      },
    ]);
  });

  it('shows every problem at once and saves nothing', async () => {
    const { user } = renderItems({
      'POST /api/v1/finance-budget-items': {
        status: 400,
        body: validationFailed([
          { field: 'budgetName', message: 'Enter a budget name.' },
          {
            field: 'code',
            message: "The code 'LUB-EU-DIG' is already used in 2026 by 'Europe – Digital'.",
          },
        ]),
      },
    });

    await screen.findByRole('region', { name: '2026 Lubricants Budget' });
    await user.click(screen.getByRole('button', { name: 'New finance budget item' }));
    const form = await screen.findByRole('dialog', { name: 'New finance budget item' });
    await user.type(within(form).getByRole('textbox', { name: /^Code/ }), 'LUB-EU-DIG');
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    expect(await within(form).findByText('2 problems need your attention')).toBeInTheDocument();
    expect(within(form).getAllByText('Enter a budget name.')).toHaveLength(2);
    expect(
      within(form).getAllByText(
        "The code 'LUB-EU-DIG' is already used in 2026 by 'Europe – Digital'.",
      ),
    ).toHaveLength(2);
  });

  it('locks the year of a saved item and saves only the changed fields', async () => {
    const { user, api } = renderItems({
      'PATCH /api/v1/finance-budget-items/FBI-1001': {
        body: { ...DIGITAL, name: 'Europe – Online', version: 3 },
      },
    });

    await chooseAction('Europe – Digital', 'Edit', user);
    const form = await screen.findByRole('dialog', { name: 'Edit finance budget item' });
    expect(within(form).queryByRole('combobox', { name: /^Budget year/ })).not.toBeInTheDocument();
    expect(form).toHaveTextContent(YEAR_LOCK);
    const name = within(form).getByRole('textbox', { name: /^Item name/ });
    await user.clear(name);
    await user.type(name, 'Europe – Online');
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    expect(await findMessages()).toHaveTextContent("Finance budget item 'Europe – Online' saved");
    expect(api.bodiesOf('PATCH /api/v1/finance-budget-items/FBI-1001')).toEqual([
      { name: 'Europe – Online', version: 2 },
    ]);
  });

  it('keeps the typed values after a conflict and saves again over it (FR-007)', async () => {
    let patches = 0;
    const { user, api } = renderItems({
      'PATCH /api/v1/finance-budget-items/FBI-1002': () => {
        patches += 1;
        return patches === 1
          ? { status: 409, body: versionConflict() }
          : { body: { ...ASIA, code: 'LUB-AS-RET' } };
      },
      'GET /api/v1/finance-budget-items/FBI-1002': { body: { ...ASIA, version: 5 } },
    });

    await chooseAction('Asia – Retail', 'Edit', user);
    const form = await screen.findByRole('dialog', { name: 'Edit finance budget item' });
    await user.type(within(form).getByRole('textbox', { name: /^Code/ }), 'LUB-AS-RET');
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    expect(await within(form).findByRole('alert')).toHaveTextContent(
      'Not saved — Daniel Okoro changed this finance budget item on 14 Mar 2026 at 10:42',
    );
    await waitFor(() =>
      expect(
        api.requests.some(
          (r) => r.method === 'GET' && r.path === '/api/v1/finance-budget-items/FBI-1002',
        ),
      ).toBe(true),
    );
    await user.click(within(form).getByRole('button', { name: 'Save' }));

    await findMessages();
    expect(api.bodiesOf('PATCH /api/v1/finance-budget-items/FBI-1002')).toEqual([
      { code: 'LUB-AS-RET', version: 1 },
      { code: 'LUB-AS-RET', version: 5 },
    ]);
  });
});

describe('FinanceBudgetItemsPage: Rename budget (US5 scenario 4, FR-015)', () => {
  it('renames the whole budget, saying what follows', async () => {
    const { user, api } = renderItems({
      'PATCH /api/v1/finance-budgets/FB-1001': {
        body: { ...LUBRICANTS_BUDGET, name: '2026 Lubricants Budget (EU)', version: 5 },
      },
    });

    await screen.findByRole('region', { name: '2026 Lubricants Budget' });
    await user.click(
      screen.getByRole('button', { name: "Rename budget '2026 Lubricants Budget'" }),
    );
    const dialog = await screen.findByRole('dialog', {
      name: "Rename budget '2026 Lubricants Budget'",
    });
    expect(dialog).toHaveTextContent(
      'All 3 items move with it. 2 plans use it as their default finance budget and will show the new name.',
    );
    const name = within(dialog).getByRole('textbox', { name: /^New name/ });
    expect(name).toHaveFocus();
    await user.clear(name);
    await user.type(name, '2026 Lubricants Budget (EU)');
    await user.click(within(dialog).getByRole('button', { name: 'Rename budget' }));

    expect(await findMessages()).toHaveTextContent(
      "Budget renamed to '2026 Lubricants Budget (EU)'",
    );
    expect(api.bodiesOf('PATCH /api/v1/finance-budgets/FB-1001')).toEqual([
      { name: '2026 Lubricants Budget (EU)', version: 4 },
    ]);
  });

  it('shows a taken name at the field and keeps the dialog open', async () => {
    const { user } = renderItems({
      'PATCH /api/v1/finance-budgets/FB-1002': {
        status: 400,
        body: validationFailed([
          { field: 'name', message: "2026 already has a budget called '2026 Lubricants Budget'." },
        ]),
      },
    });

    await screen.findByRole('region', { name: '2026 Mobility Budget' });
    await user.click(screen.getByRole('button', { name: "Rename budget '2026 Mobility Budget'" }));
    const dialog = await screen.findByRole('dialog', {
      name: "Rename budget '2026 Mobility Budget'",
    });
    expect(dialog).toHaveTextContent('All 1 item moves with it.');
    const name = within(dialog).getByRole('textbox', { name: /^New name/ });
    await user.clear(name);
    await user.type(name, '2026 Lubricants Budget');
    await user.click(within(dialog).getByRole('button', { name: 'Rename budget' }));

    expect(
      await within(dialog).findByText("2026 already has a budget called '2026 Lubricants Budget'."),
    ).toBeInTheDocument();
    expect(name).toHaveValue('2026 Lubricants Budget');
  });
});

describe('FinanceBudgetItemsPage: status and delete (US5 scenario 5, FR-004)', () => {
  it('moves an unused item to the recycle bin and offers Undo', async () => {
    const { user } = renderItems({
      'DELETE /api/v1/finance-budget-items/FBI-1002': {
        body: deleted('finance budget item', 'FBI-1002', 'Asia – Retail'),
      },
    });

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
    const { user, api } = renderItems({
      'POST /api/v1/finance-budget-items/FBI-1001/status': { body: { ...DIGITAL, active: false } },
    });

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
    const { user, api } = renderItems(
      { 'POST /api/v1/finance-budget-items/FBI-1003/status': { body: { ...OLD, active: true } } },
      '/settings/finance-budget-items?status=inactive',
    );

    await chooseAction('Old item', 'Reactivate', user);

    expect(await findMessages()).toHaveTextContent("Finance budget item 'Old item' reactivated");
    expect(api.bodiesOf('POST /api/v1/finance-budget-items/FBI-1003/status')).toEqual([
      { active: true, version: 1 },
    ]);
  });
});
