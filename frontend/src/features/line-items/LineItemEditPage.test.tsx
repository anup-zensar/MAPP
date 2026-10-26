import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/testing/mockApi';
import type { MockHandler } from '@/testing/mockApi';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { LineItemEditPage } from './LineItemEditPage';
import {
  approvedActivity,
  details,
  DIGITAL,
  eur,
  EUR,
  formAnswers,
  formOptions,
  lineItem,
  notFound,
  spreadMonth,
  usd,
} from '@/testing/fixtures/line-items';

const GET = 'GET /api/v1/line-items/L-1001';
const PATCH = 'PATCH /api/v1/line-items/L-1001';
const OPTIONS = 'GET /api/v1/activities/A-1001/line-items/form-options';

function renderPage(answers: Record<string, MockHandler> = {}, route = '/line-items/L-1001/edit') {
  const api = mockApi({
    ...formAnswers(),
    [GET]: { body: details() },
    [OPTIONS]: { body: formOptions() },
    ...answers,
  });
  return {
    api,
    ...renderWithProviders(<LineItemEditPage />, {
      route,
      path: '/line-items/:lineItemId/edit',
      user: testUser(),
    }),
  };
}

function approved(): Record<string, MockHandler> {
  return {
    [GET]: { body: details({ activity: approvedActivity() }) },
    [OPTIONS]: { body: formOptions({ activity: approvedActivity() }) },
  };
}

function button(name: string) {
  return screen.getByText(name, { selector: 'button' });
}

describe('LineItemEditPage (US4, FR-027)', () => {
  it('opens with the stored values', async () => {
    renderPage();

    expect(await screen.findByLabelText('Line item name')).toHaveValue('Agency fee');
    expect(screen.getByRole('heading', { level: 1, name: 'Edit line item' })).toBeInTheDocument();
    expect(document.title).toBe('Edit Agency fee · MAPP');
    expect(screen.getByLabelText('Planned cost')).toHaveValue('50,000.00');
    expect(screen.queryByText('Save and add another')).not.toBeInTheDocument();
    expect(screen.getByLabelText('Activity summary')).toHaveTextContent(
      'Unassigned now20,000.00 USD',
    );
  });

  it('saves coding changes of an approved activity without a dialog (scenario 2, FR-032)', async () => {
    const { user, api, router } = renderPage({
      ...approved(),
      [PATCH]: { body: lineItem({ financeBudgetItem: DIGITAL }) },
    });
    await screen.findByLabelText('Line item name');

    await user.selectOptions(
      screen.getByLabelText('Finance budget item'),
      'Europe – Digital · 2026 Lubricants Budget · LUB-EU-DIG',
    );
    await user.click(screen.getByRole('radio', { name: 'All in last month' }));
    await user.click(button('Save'));

    await waitFor(() => expect(router.state.location.pathname).toBe('/line-items/L-1001'));
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    expect(api.bodiesOf(PATCH)[0]).toMatchObject({
      financeBudgetItemId: DIGITAL.id,
      spread: { method: 'lastMonth', months: [] },
      restartApproval: false,
      version: 1,
    });
    expect(await screen.findByText("Line item 'Agency fee' saved")).toBeInTheDocument();
  });

  it('asks before a cost change restarts approval (scenario 3, FR-031)', async () => {
    const { user, api } = renderPage({ ...approved(), [PATCH]: { body: lineItem() } });
    const cost = await screen.findByLabelText('Planned cost');

    await user.clear(cost);
    await user.type(cost, '55,000.00');
    expect(screen.getByLabelText('Activity summary')).toHaveTextContent(
      'After saving15,000.00 USD',
    );
    await user.click(button('Save'));
    let dialog = await screen.findByRole('alertdialog', {
      name: 'This activity will need approval again',
    });
    expect(dialog).toHaveTextContent("Planned cost of 'Agency fee': 50,000.00 USD → 55,000.00 USD");

    await user.click(within(dialog).getByRole('button', { name: 'Keep editing' }));
    await waitFor(() => expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument());
    expect(screen.getByLabelText('Planned cost')).toHaveValue('55,000.00');
    expect(api.bodiesOf(PATCH)).toHaveLength(0);

    await user.click(button('Save'));
    dialog = await screen.findByRole('alertdialog', {
      name: 'This activity will need approval again',
    });
    await user.click(within(dialog).getByRole('button', { name: 'Save and restart approval' }));
    await waitFor(() => expect(api.bodiesOf(PATCH)).toHaveLength(1));
    expect(api.bodiesOf(PATCH)[0]).toMatchObject({
      plannedCost: '55000.00',
      restartApproval: true,
    });
  });

  it('asks before a currency change restarts approval', async () => {
    const { user } = renderPage(approved());
    await screen.findByLabelText('Line item name');

    await user.selectOptions(screen.getByLabelText('Currency'), 'EUR');
    await user.click(button('Save'));

    expect(
      await screen.findByRole('alertdialog', { name: 'This activity will need approval again' }),
    ).toHaveTextContent("Planned cost of 'Agency fee': 50,000.00 USD → 50,000.00 EUR");
  });

  it('keeps the values after a conflict and saves with the new version next', async () => {
    let gets = 0;
    let patches = 0;
    const { user, api } = renderPage({
      [GET]: () => ({ body: details({ lineItem: lineItem({ version: gets++ === 0 ? 1 : 2 }) }) }),
      [PATCH]: () =>
        patches++ === 0
          ? {
              status: 409,
              body: {
                error: {
                  code: 'version_conflict',
                  message: 'Someone else changed this line item.',
                  fields: [],
                  conflict: { changedBy: 'Daniel Okoro', changedAt: '2026-03-14T10:30:00Z' },
                },
              },
            }
          : { body: lineItem({ name: 'Agency fees' }) },
    });
    const name = await screen.findByLabelText('Line item name');

    await user.type(name, 's');
    await user.click(button('Save'));

    expect(
      await screen.findByText(/^Not saved — Daniel Okoro changed this line item/),
    ).toBeInTheDocument();
    expect(screen.getByLabelText('Line item name')).toHaveValue('Agency fees');
    await user.click(button('Save'));
    await waitFor(() => expect(api.bodiesOf(PATCH)).toHaveLength(2));
    expect(api.bodiesOf(PATCH).map((body) => (body as { version: number }).version)).toEqual([
      1, 2,
    ]);
  });

  it('says when the activity was deleted while editing (edge case, D-154)', async () => {
    const message =
      'This activity was deleted — restore it from the recycle bin to keep working on its line items.';
    const { user } = renderPage({ [PATCH]: notFound(message) });
    await screen.findByLabelText('Line item name');

    await user.click(button('Save'));

    expect(await screen.findByText(message)).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Go to the recycle bin' })).toBeInTheDocument();
  });

  it('says when the line item is gone on opening', async () => {
    renderPage({
      [GET]: notFound('This line item no longer exists — someone may have deleted it.'),
    });

    expect(
      await screen.findByText('This line item no longer exists — someone may have deleted it.'),
    ).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Go to Plans' })).toBeInTheDocument();
  });

  it('focuses the field of a completeness link (?field=)', async () => {
    renderPage({}, '/line-items/L-1001/edit?field=financeBudgetItemId');

    await waitFor(() => expect(screen.getByLabelText('Finance budget item')).toHaveFocus());
  });

  it("shows a saved custom spread that doesn't add up and uses its total (US3 scenario 3)", async () => {
    const custom = lineItem({
      currency: EUR,
      plannedCost: eur('10000.00'),
      plannedCostUsd: usd('10869.57'),
      spread: {
        method: 'custom',
        state: 'spread',
        months: [
          spreadMonth('2026-02', eur('2000.00'), usd('2173.91'), true),
          spreadMonth('2026-03', eur('3333.33'), usd('3623.18')),
          spreadMonth('2026-04', eur('3333.33'), usd('3623.18')),
          spreadMonth('2026-05', eur('3333.34'), usd('3623.20')),
        ],
        customTotal: eur('12000.00'),
        addsUp: false,
      },
      missing: [{ field: 'spread', label: "Spread doesn't add up to the planned cost" }],
    });
    const { user, api } = renderPage({
      [GET]: { body: details({ lineItem: custom }) },
      [PATCH]: { body: custom },
    });
    await screen.findByLabelText('Line item name');

    expect(screen.getByRole('radio', { name: 'Custom' })).toBeChecked();
    expect(screen.getByLabelText('Amount for Feb 2026')).toHaveValue('2,000.00');
    expect(
      screen.getAllByText("Outside the activity's dates — allowed, e.g. a deposit").length,
    ).toBeGreaterThan(0);
    expect(
      screen.getByText('12,000.00 of 10,000.00 EUR spread — 2,000.00 EUR too much'),
    ).toBeInTheDocument();

    await user.click(button('Use month total as planned cost'));
    expect(screen.getByLabelText('Planned cost')).toHaveValue('12,000.00');
    expect(screen.getByText('12,000.00 of 12,000.00 EUR spread — adds up')).toBeInTheDocument();
    await user.click(button('Save'));
    await waitFor(() => expect(api.bodiesOf(PATCH)).toHaveLength(1));
    expect(api.bodiesOf(PATCH)[0]).toMatchObject({
      plannedCost: '12000.00',
      spread: {
        method: 'custom',
        months: [
          { month: '2026-02', amount: '2000.00' },
          { month: '2026-03', amount: '3333.33' },
          { month: '2026-04', amount: '3333.33' },
          { month: '2026-05', amount: '3333.34' },
        ],
      },
    });
  });
});
