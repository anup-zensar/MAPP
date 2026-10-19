import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import { FinanceBudgetFields } from './FinanceBudgetFields';
import { changeBudgetYear, newFormState, withField } from './lineItemFormState';
import type { LineItemFormState } from './lineItemFormState';
import type { FinanceBudgetItemRef } from './lineItemsApi';
import {
  activity,
  BRAND_INACTIVE,
  DIGITAL,
  DIGITAL_2027,
  EVENTS,
  formOptions,
  MOBILITY_RETAIL,
} from './testData';

const ITEMS = [DIGITAL, EVENTS, MOBILITY_RETAIL, DIGITAL_2027];

function Harness({
  start,
  items = ITEMS,
  years = [2026, 2027],
  planActivity = activity(),
}: {
  start: LineItemFormState;
  items?: FinanceBudgetItemRef[];
  years?: number[];
  planActivity?: ReturnType<typeof activity>;
}) {
  const [state, setState] = useState(start);
  return (
    <FinanceBudgetFields
      state={state}
      activity={planActivity}
      years={years}
      items={items}
      problems={[]}
      onYearChange={(year) => setState((s) => changeBudgetYear(s, year, items))}
      onItemChange={(id) => setState((s) => withField(s, 'financeBudgetItemId', id))}
    />
  );
}

describe('FinanceBudgetFields (FR-015, FR-016)', () => {
  it("offers the years and that year's items, the plan's default budget first", () => {
    render(<Harness start={newFormState(formOptions())} />);

    const year = screen.getByLabelText('Finance budget year');
    expect([...year.querySelectorAll('option')].map((o) => o.textContent)).toEqual([
      'Choose…',
      '2026',
      '2027',
    ]);
    expect(screen.getByText("The plan's year")).toBeInTheDocument();
    const item = screen.getByLabelText('Finance budget item');
    expect([...item.querySelectorAll('option')].map((o) => o.textContent)).toEqual([
      'Choose…',
      'Europe – Digital · 2026 Lubricants Budget · LUB-EU-DIG',
      'Europe – Events · 2026 Lubricants Budget · LUB-EU-EVT',
      'China – Retail · 2026 Mobility Budget · MOB-CN-RET',
    ]);
    expect(
      screen.getByText(
        "Items of '2026 Lubricants Budget', the plan's default finance budget, are listed first.",
      ),
    ).toBeInTheDocument();
  });

  it('says when the item is pre-selected (R12)', () => {
    const options = formOptions();
    options.defaults = {
      ...options.defaults,
      financeBudgetItemId: DIGITAL.id,
      financeBudgetItemPreselected: true,
    };
    render(<Harness start={newFormState(options)} />);

    expect(screen.getByLabelText('Finance budget item')).toHaveValue(DIGITAL.id);
    expect(
      screen.getByText("Pre-selected — the only active item of '2026 Lubricants Budget'"),
    ).toBeInTheDocument();
  });

  it('says which item a year change clears, and brings it back with its year', async () => {
    const user = userEvent.setup();
    const start = withField(newFormState(formOptions()), 'financeBudgetItemId', DIGITAL.id);
    render(<Harness start={start} />);

    await user.selectOptions(screen.getByLabelText('Finance budget year'), '2027');
    expect(
      screen.getByText(
        "Finance budget item 'Europe – Digital' will be cleared — it belongs to 2026.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByLabelText('Finance budget item')).toHaveValue('');

    await user.selectOptions(screen.getByLabelText('Finance budget year'), '2026');
    expect(screen.getByLabelText('Finance budget item')).toHaveValue(DIGITAL.id);
  });

  it('says when a year has no items yet', () => {
    const start = { ...newFormState(formOptions()), financeBudgetYear: 2028 };
    render(<Harness start={start} years={[2026, 2028]} />);

    expect(screen.getByLabelText('Finance budget item')).toBeDisabled();
    expect(
      screen.getByText(
        'No finance budget items for 2028 yet — an administrator adds them in Settings.',
      ),
    ).toBeInTheDocument();
  });

  it('keeps an inactive linked item and names it as a problem (FR-016)', () => {
    const start = withField(newFormState(formOptions()), 'financeBudgetItemId', BRAND_INACTIVE.id);
    render(<Harness start={start} items={[...ITEMS, BRAND_INACTIVE]} />);

    expect(
      screen.getByRole('option', {
        name: 'Europe – Brand · 2026 Lubricants Budget · LUB-EU-BRD (inactive)',
      }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        "Choose an active finance budget item — 'Europe – Brand' is no longer active.",
      ),
    ).toBeInTheDocument();
  });

  it('is searchable on item, budget or code when there are many items', () => {
    const many = Array.from({ length: 12 }, (_, index) => ({
      ...DIGITAL,
      id: `FBI-9${index}`,
      name: `Item ${index}`,
    }));
    render(<Harness start={newFormState(formOptions())} items={many} />);

    expect(screen.getByText(/Search item, budget or code/)).toBeInTheDocument();
  });
});
