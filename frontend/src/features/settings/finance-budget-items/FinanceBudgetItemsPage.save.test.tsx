import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { findMessages } from '@/testing/render';
import {
  ASIA,
  DIGITAL,
  item,
  LUBRICANTS_BUDGET,
  YEAR_LOCK,
} from '@/testing/fixtures/finance-budget-items';
import { validationFailed, versionConflict } from '@/testing/fixtures/settings';
import { chooseAction, renderFinanceBudgetItemsPage } from '@/testing/fixtures/settings-pages';
import { reply } from '@/testing/replies';

describe('FinanceBudgetItemsPage: create and edit (US5 scenarios 3 and 4, FR-014)', () => {
  it('creates "Europe – Retail" in an existing budget, offering its name while typing', async () => {
    const created = item({ id: 'FBI-1005', name: 'Europe – Retail', code: 'LUB-EU-RET' });
    const { user, api } = renderFinanceBudgetItemsPage([
      reply('POST /api/v1/finance-budget-items', { status: 201, body: created }),
    ]);

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
    await user.click(budgetName);
    await user.paste('2026 Lubricants Budget');
    await user.click(within(form).getByRole('textbox', { name: /^Item name/ }));
    await user.paste('Europe – Retail');
    await user.click(within(form).getByRole('textbox', { name: /^Code/ }));
    await user.paste('LUB-EU-RET');
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
    const { user } = renderFinanceBudgetItemsPage([
      reply('POST /api/v1/finance-budget-items', {
        status: 400,
        body: validationFailed([
          { field: 'budgetName', message: 'Enter a budget name.' },
          {
            field: 'code',
            message: "The code 'LUB-EU-DIG' is already used in 2026 by 'Europe – Digital'.",
          },
        ]),
      }),
    ]);

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
    const { user, api } = renderFinanceBudgetItemsPage([
      reply(
        'PATCH /api/v1/finance-budget-items/{itemId}',
        {
          body: { ...DIGITAL, name: 'Europe – Online', version: 3 },
        },
        { itemId: 'FBI-1001' },
      ),
    ]);

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
    const { user, api } = renderFinanceBudgetItemsPage([
      reply(
        'PATCH /api/v1/finance-budget-items/{itemId}',
        () => {
          patches += 1;
          return patches === 1
            ? { status: 409, body: versionConflict() }
            : { body: { ...ASIA, code: 'LUB-AS-RET' } };
        },
        { itemId: 'FBI-1002' },
      ),
      reply(
        'GET /api/v1/finance-budget-items/{itemId}',
        { body: { ...ASIA, version: 5 } },
        { itemId: 'FBI-1002' },
      ),
    ]);

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
    const { user, api } = renderFinanceBudgetItemsPage([
      reply(
        'PATCH /api/v1/finance-budgets/{budgetId}',
        {
          body: { ...LUBRICANTS_BUDGET, name: '2026 Lubricants Budget (EU)', version: 5 },
        },
        { budgetId: 'FB-1001' },
      ),
    ]);

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
    await user.paste('2026 Lubricants Budget (EU)');
    await user.click(within(dialog).getByRole('button', { name: 'Rename budget' }));

    expect(await findMessages()).toHaveTextContent(
      "Budget renamed to '2026 Lubricants Budget (EU)'",
    );
    expect(api.bodiesOf('PATCH /api/v1/finance-budgets/FB-1001')).toEqual([
      { name: '2026 Lubricants Budget (EU)', version: 4 },
    ]);
  });

  it('shows a taken name at the field and keeps the dialog open', async () => {
    const { user } = renderFinanceBudgetItemsPage([
      reply(
        'PATCH /api/v1/finance-budgets/{budgetId}',
        {
          status: 400,
          body: validationFailed([
            {
              field: 'name',
              message: "2026 already has a budget called '2026 Lubricants Budget'.",
            },
          ]),
        },
        { budgetId: 'FB-1002' },
      ),
    ]);

    await screen.findByRole('region', { name: '2026 Mobility Budget' });
    await user.click(screen.getByRole('button', { name: "Rename budget '2026 Mobility Budget'" }));
    const dialog = await screen.findByRole('dialog', {
      name: "Rename budget '2026 Mobility Budget'",
    });
    expect(dialog).toHaveTextContent('All 1 item moves with it.');
    const name = within(dialog).getByRole('textbox', { name: /^New name/ });
    await user.clear(name);
    await user.paste('2026 Lubricants Budget');
    await user.click(within(dialog).getByRole('button', { name: 'Rename budget' }));

    expect(
      await within(dialog).findByText("2026 already has a budget called '2026 Lubricants Budget'."),
    ).toBeInTheDocument();
    expect(name).toHaveValue('2026 Lubricants Budget');
  });
});
