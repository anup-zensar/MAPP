import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Tooltip } from 'radix-ui';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { EUR, eur, lineItem, spreadMonth, usd } from '@/testing/fixtures/line-items';
import { formOptions } from '@/testing/fixtures/line-item-forms';
import {
  addCustomMonth,
  changeCustomMonth,
  changePlannedCost,
  changeSpreadMethod,
  editFormState,
  newFormState,
} from './line-item-form-state';
import type { LineItemFormState } from './line-item-form-state';
import type { ActivityDates } from './line-item-math';
import { SpreadEditor } from './SpreadEditor';

const MAR_MAY: ActivityDates = { startDate: '2026-03-01', endDate: '2026-05-31', ongoing: false };

function tenThousandEuros(): LineItemFormState {
  return { ...newFormState(formOptions()), currency: 'EUR', plannedCost: '10,000.00' };
}

function Harness({ start, dates = MAR_MAY }: { start: LineItemFormState; dates?: ActivityDates }) {
  const [state, setState] = useState(start);
  return (
    <Tooltip.Provider>
      <SpreadEditor
        method={state.spreadMethod}
        plannedCost={state.plannedCost}
        currency={state.currency}
        dates={dates}
        customMonths={state.customMonths}
        customTyped={state.customTyped}
        source={state.sources.spread}
        problems={[]}
        onMethodChange={(method) => setState((s) => changeSpreadMethod(s, method, dates, 2026))}
        onMonthChange={(month, amount) => setState((s) => changeCustomMonth(s, month, amount))}
        onAddMonth={(month) => setState((s) => addCustomMonth(s, month))}
        onUseTotal={(amount) => setState((s) => changePlannedCost(s, amount))}
      />
      <output aria-label="Planned cost now">{state.plannedCost}</output>
    </Tooltip.Provider>
  );
}

function preview() {
  return screen.getByRole('list', { name: 'Preview of the spread' }).textContent;
}

describe('038 FR-042 (L-7): pasting a row of amounts', () => {
  it('fills the month and the ones after it, and says what did not fit', async () => {
    const user = userEvent.setup();
    const start = changeSpreadMethod(tenThousandEuros(), 'custom', MAR_MAY, 2026);
    renderWithProviders(<Harness start={start} />);

    await user.click(screen.getByRole('textbox', { name: /Mar 2026/ }));
    await user.paste('1000\t2000\t3000\t4000');

    expect(screen.getByRole('textbox', { name: /Mar 2026/ })).toHaveValue('1000.00');
    expect(screen.getByRole('textbox', { name: /Apr 2026/ })).toHaveValue('2000.00');
    expect(screen.getByRole('textbox', { name: /May 2026/ })).toHaveValue('3000.00');
    expect(
      screen.getByText('1 pasted value was left out: there is no month after May 2026.'),
    ).toBeInTheDocument();
  });
});

describe('SpreadEditor (US3, FR-017 – FR-021)', () => {
  it('previews Evenly, All in first month and All in last month (scenario 1)', async () => {
    const user = userEvent.setup();
    renderWithProviders(<Harness start={tenThousandEuros()} />);

    expect(screen.getByRole('radio', { name: 'Evenly' })).toBeChecked();
    expect(screen.getByText('Mar 2026 · 3,333.33 EUR')).toBeInTheDocument();
    expect(screen.getByText('Apr 2026 · 3,333.33 EUR')).toBeInTheDocument();
    expect(screen.getByText('May 2026 · 3,333.34 EUR')).toBeInTheDocument();

    await user.click(screen.getByRole('radio', { name: 'All in first month' }));
    expect(preview()).toBe('Mar 2026 · 10,000.00 EUR');
    await user.click(screen.getByRole('radio', { name: 'All in last month' }));
    expect(preview()).toBe('May 2026 · 10,000.00 EUR');
  });

  it('starts Custom from the spread, adds a month before and totals it (scenario 2, FR-018)', async () => {
    const user = userEvent.setup();
    renderWithProviders(<Harness start={tenThousandEuros()} />);

    await user.click(screen.getByRole('radio', { name: 'Custom' }));
    expect(screen.getByLabelText('Amount for Mar 2026')).toHaveValue('3,333.33');
    expect(screen.getByLabelText('Amount for May 2026')).toHaveValue('3,333.34');
    expect(screen.getByText('10,000.00 of 10,000.00 EUR spread — adds up')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Add month before' }));
    const february = screen.getByLabelText('Amount for Feb 2026');
    await user.type(february, '2000');
    expect(
      screen.getByText("Outside the activity's dates — allowed, e.g. a deposit"),
    ).toBeInTheDocument();
    expect(
      screen.getByText('12,000.00 of 10,000.00 EUR spread — 2,000.00 EUR too much'),
    ).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Use month total as planned cost' }));
    expect(screen.getByLabelText('Planned cost now')).toHaveTextContent('12,000.00');
    expect(screen.getByText('12,000.00 of 12,000.00 EUR spread — adds up')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Add month after' }));
    expect(screen.getByLabelText('Amount for Jun 2026')).toHaveValue('');
    await user.clear(february);
    expect(
      screen.getByText('10,000.00 of 12,000.00 EUR spread — 2,000.00 EUR left'),
    ).toBeInTheDocument();
  });

  it('says that an ongoing activity puts everything in its first month (scenario 4, FR-021)', async () => {
    const user = userEvent.setup();
    const ongoing = { startDate: '2026-03-15', endDate: null, ongoing: true };
    renderWithProviders(<Harness start={tenThousandEuros()} dates={ongoing} />);

    expect(
      screen.getByText(
        "This activity is ongoing (no end date), so 'Evenly' puts everything in the first month — choose Custom to spread it",
      ),
    ).toBeInTheDocument();
    expect(preview()).toBe('Mar 2026 · 10,000.00 EUR');
    await user.click(screen.getByRole('radio', { name: 'All in last month' }));
    expect(
      screen.getByText(
        "This activity is ongoing (no end date), so 'All in last month' puts everything in the first month — choose Custom to spread it",
      ),
    ).toBeInTheDocument();
    await user.click(screen.getByRole('radio', { name: 'Custom' }));
    expect(screen.getByLabelText('Amount for Dec 2026')).toBeInTheDocument();
  });

  it('says when the activity has no dates yet (FR-017)', () => {
    renderWithProviders(
      <Harness
        start={tenThousandEuros()}
        dates={{ startDate: null, endDate: null, ongoing: false }}
      />,
    );

    expect(
      screen.getByText("Not spread yet — add the activity's dates to see the spread"),
    ).toBeInTheDocument();
  });

  it('says that leaving Custom replaces the typed amounts (scenario 6, D-151)', async () => {
    const user = userEvent.setup();
    const custom = editFormState(
      lineItem({
        currency: EUR,
        plannedCost: eur('10000.00'),
        spread: {
          method: 'custom',
          state: 'spread',
          months: [spreadMonth('2026-03', eur('10000.00'), usd('10869.57'))],
          customTotal: eur('10000.00'),
          addsUp: true,
        },
      }),
    );
    renderWithProviders(<Harness start={custom} />);

    expect(
      screen.queryByText(
        'The month amounts you typed will be replaced by this spread when you save.',
      ),
    ).not.toBeInTheDocument();
    await user.click(screen.getByRole('radio', { name: 'Evenly' }));
    expect(
      screen.getByText(
        'The month amounts you typed will be replaced by this spread when you save.',
      ),
    ).toBeInTheDocument();
  });
});
