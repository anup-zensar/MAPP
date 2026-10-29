import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { BulkBar, useSelection } from '@/features/bulk';
import { expectNoAxeViolations } from '@/testing/axe';
import { gridResult, standard2026 } from '@/testing/fixtures/all-work';
import { testUser } from '@/testing/fixtures/users';
import { renderWithProviders } from '@/testing/render';
import { AllWorkGrid } from './AllWorkGrid';
import { gridSelection, matchingRecords } from './grid-selection';
import { continuedPath, pageOf, visibleRows } from './view-rows';

function Harness({ pageSize = 50 }: { pageSize?: number }) {
  const selection = useSelection();
  const result = gridResult();
  const state = standard2026();
  const page = pageOf(visibleRows(result.rows, state), 1, pageSize);
  return (
    <>
      <AllWorkGrid
        result={result}
        rows={page.rows}
        continued={continuedPath(page.rows, result.rows)}
        search=""
        sort={null}
        onSort={() => undefined}
        onToggle={() => undefined}
        onOpen={() => undefined}
        selection={gridSelection(
          selection,
          page.rows.map((v) => v.row),
        )}
      />
      <BulkBar selection={selection} matching={matchingRecords(result.rows)} />
    </>
  );
}

function bar() {
  return screen.getByRole('region', { name: 'Selection' });
}

describe('012 FR-001 – FR-005: ticking rows of the All work grid', () => {
  it('shows a checkbox on campaign, initiative and activity rows only, and a campaign selects only itself', async () => {
    const { user } = renderWithProviders(<Harness />, { user: testUser() });

    expect(screen.queryByRole('checkbox', { name: /Select plan/ })).not.toBeInTheDocument();
    await user.click(screen.getByRole('checkbox', { name: "Select campaign 'Helix launch'" }));

    expect(within(bar()).getByText('1 selected · 1 campaign')).toBeInTheDocument();
    expect(
      screen.getByRole('checkbox', { name: "Select initiative 'Helix digital'" }),
    ).not.toBeChecked();
    for (const action of [
      'Change owner',
      'Change field',
      'Submit for approval',
      'Copy to…',
      'Move to…',
      'Delete',
    ]) {
      expect(within(bar()).getByRole('button', { name: action })).toBeInTheDocument();
    }
    await expectNoAxeViolations();
  });

  it('selects the matching rows of the page with the header checkbox, then all matching, then clears', async () => {
    const { user } = renderWithProviders(<Harness pageSize={4} />, { user: testUser() });

    await user.click(screen.getByRole('checkbox', { name: 'Select all rows on this page' }));

    expect(
      within(bar()).getByText(/^3 selected · 1 campaign, 1 initiative, 1 activity$/),
    ).toBeInTheDocument();
    await user.click(within(bar()).getByRole('button', { name: /Select all \d+ matching/ }));
    expect(within(bar()).queryByRole('button', { name: /Select all/ })).not.toBeInTheDocument();
    await user.click(within(bar()).getByRole('button', { name: 'Clear' }));
    expect(screen.queryByRole('region', { name: 'Selection' })).not.toBeInTheDocument();
  });

  it('ticks a range with Shift', async () => {
    const { user } = renderWithProviders(<Harness />, { user: testUser() });

    await user.click(screen.getByRole('checkbox', { name: "Select campaign 'Helix launch'" }));
    await user.keyboard('{Shift>}');
    await user.click(screen.getByRole('checkbox', { name: "Select activity 'Helix social ads'" }));
    await user.keyboard('{/Shift}');

    expect(
      within(bar()).getByText('4 selected · 1 campaign, 1 initiative, 2 activities'),
    ).toBeInTheDocument();
  });
});
