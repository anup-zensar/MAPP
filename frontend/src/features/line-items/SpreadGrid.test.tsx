import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Tooltip } from 'radix-ui';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { launchFilmSection, usd } from '@/testing/fixtures/line-items';
import { SpreadGrid } from './SpreadGrid';

const KEY = 'mapp.lineItemGrid';

function renderGrid(grid = launchFilmSection().grid, noDates = false) {
  return render(
    <Tooltip.Provider>
      <SpreadGrid grid={grid} noDates={noDates} />
    </Tooltip.Provider>,
  );
}

function headers() {
  return within(screen.getByRole('table'))
    .getAllByRole('columnheader')
    .map((header) => header.textContent);
}

afterEach(() => {
  localStorage.clear();
  vi.restoreAllMocks();
});

describe('SpreadGrid (FR-003, FR-020)', () => {
  it('sums the months into quarters and remembers the choice (scenario 2)', async () => {
    const user = userEvent.setup();
    const grid = launchFilmSection().grid;
    renderGrid(grid);

    expect(headers()).toEqual(['Line item', 'Mar 2026', 'Apr 2026', 'May 2026', 'Total']);
    await user.click(screen.getByRole('button', { name: 'Quarterly' }));

    expect(headers()).toEqual(['Line item', 'Q1 2026', 'Q2 2026', 'Total']);
    expect(screen.getByRole('row', { name: /^Agency fee/ })).toHaveTextContent(
      'Agency fee16,666.6633,333.3450,000.00',
    );
    expect(localStorage.getItem(KEY)).toBe('quarterly');
  });

  it('starts with the remembered choice, and works when storage is blocked', () => {
    localStorage.setItem(KEY, 'quarterly');
    renderGrid();
    expect(headers()).toContain('Q1 2026');
  });

  it('works when storage cannot be read', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('blocked');
    });
    renderGrid();
    expect(headers()).toContain('Mar 2026');
  });

  it('marks months outside the dates and shows the No dates yet column', () => {
    const grid = {
      columns: [
        { key: '2026-02', month: '2026-02', outsideActivityDates: true },
        { key: '2026-03', month: '2026-03', outsideActivityDates: false },
        { key: 'noDates', month: null, outsideActivityDates: false },
      ],
      rows: [
        {
          lineItemId: 'L-1',
          name: 'Deposit',
          cells: [usd('2000.00'), null, null],
          total: usd('2000.00'),
        },
        {
          lineItemId: 'L-2',
          name: 'Fee',
          cells: [null, null, usd('500.00')],
          total: usd('500.00'),
        },
      ],
      totals: [usd('2000.00'), null, usd('500.00')],
      total: usd('2500.00'),
    };
    renderGrid(grid, true);

    expect(headers()).toEqual([
      'Line item',
      'Feb 2026Outside dates',
      'Mar 2026',
      'No dates yet',
      'Total',
    ]);
    expect(
      screen.getByRole('button', { name: 'About Feb 2026 outside the dates' }),
    ).toBeInTheDocument();
    expect(screen.getByRole('row', { name: /^Fee/ })).toHaveTextContent('Fee——500.00500.00');
    expect(
      screen.getByText("Add the activity's dates to spread its line items over months."),
    ).toBeInTheDocument();
  });

  it('keeps the line-item names in a sticky first column, scrolling inside the card', () => {
    renderGrid();
    const name = screen.getByRole('rowheader', { name: 'Agency fee' });
    expect(name.className).toMatch(/sticky/);
    expect(
      screen.getByRole('region', { name: 'Line items spread over time, in USD' }),
    ).toHaveAttribute('tabindex', '0');
  });
});
