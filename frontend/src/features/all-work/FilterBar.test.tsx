import { screen, waitFor, within } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import type { AllWorkResult, ViewState } from './allWorkTypes';
import { FilterBar } from './FilterBar';
import { allWorkOptions, standard2026 } from './testData';

/** The filter bar with the state kept as the page keeps it; the state is shown as JSON. */
function Harness({
  initial = standard2026(),
  count = { shown: 3, total: 3, noun: 'activities' },
  updating = false,
}: {
  initial?: ViewState;
  count?: AllWorkResult['count'];
  updating?: boolean;
}) {
  const [state, setState] = useState(initial);
  return (
    <>
      <FilterBar
        state={state}
        options={allWorkOptions()}
        count={count}
        updating={updating}
        onChange={(change) => setState(change)}
      />
      <output aria-label="State">{JSON.stringify(state)}</output>
    </>
  );
}

function renderBar(props: Parameters<typeof Harness>[0] = {}) {
  return renderWithProviders(<Harness {...props} />, { user: testUser() });
}

function current(): ViewState {
  return JSON.parse(screen.getByRole('status', { name: 'State' }).textContent ?? '{}');
}

describe('FilterBar (US2)', () => {
  it('offers exactly the filters of FR-004 and nothing else (FR-004, FR-010)', () => {
    renderBar();

    const filters = screen.getByRole('group', { name: 'Filters' });
    expect(
      within(filters)
        .getAllByRole('button')
        .map((button) => button.textContent),
    ).toEqual([
      'Year · 2026',
      'Plan',
      'Plan status',
      'Campaign / initiative status',
      'Activity status',
      'Owner',
      'Team',
      'Business unit',
      'Activity type',
      'Strategic / AOP plans',
      'Show levels',
    ]);
    expect(screen.getByRole('searchbox', { name: 'Search' })).toHaveAttribute(
      'placeholder',
      'Search name, ID, external name or description',
    );
  });

  it('applies a filter as soon as it is ticked and shows it as a chip (FR-008, FR-009)', async () => {
    const { user } = renderBar();

    await user.click(screen.getByRole('button', { name: 'Activity status' }));
    const menu = await screen.findByRole('dialog', { name: 'Activity status' });
    expect(
      within(menu)
        .getAllByRole('checkbox')
        .map((box) => box.closest('label')?.textContent),
    ).toEqual(['Created', 'Submitted', 'Declined', 'Approved', 'In progress', 'Complete']);
    await user.click(within(menu).getByRole('checkbox', { name: 'Declined' }));

    expect(current().filters.activityStatuses).toEqual(['declined']);
    expect(
      screen.getByRole('button', { name: 'Remove filter: Activity status: Declined' }),
    ).toHaveTextContent('Activity status: Declined');
    expect(screen.getByRole('button', { name: 'Activity status · 1' })).toBeInTheDocument();
  });

  it('keeps the name of a chosen person with the value (FR-031)', async () => {
    const { user } = renderBar();

    await user.click(screen.getByRole('button', { name: 'Owner' }));
    const menu = await screen.findByRole('dialog', { name: 'Owner' });
    expect(within(menu).getByText(/Tom Weber/)).toHaveTextContent('Tom Weber (inactive)');
    await user.click(within(menu).getByRole('checkbox', { name: 'Daniel Okoro' }));

    expect(current().filters.owners).toEqual([{ id: 'U-1003', label: 'Daniel Okoro' }]);
    expect(
      screen.getByRole('button', { name: 'Remove filter: Owner: Daniel Okoro' }),
    ).toBeVisible();
  });

  it('puts My teams first in the Team filter (FR-004)', async () => {
    const { user } = renderBar();

    await user.click(screen.getByRole('button', { name: 'Team' }));
    const menu = await screen.findByRole('dialog', { name: 'Team' });
    const boxes = within(menu).getAllByRole('checkbox');
    expect(boxes.map((box) => box.closest('label')?.textContent)).toEqual([
      'My teams',
      'Lubricants Europe',
      'Global Marketing Ops',
      'Mobility China',
    ]);
    await user.click(boxes[0]);

    expect(current().filters.teams).toEqual([{ id: 'T-1001', label: 'Lubricants Europe' }]);
  });

  it('groups the campaign / initiative statuses by Approval progress and Timing', async () => {
    const { user } = renderBar();

    await user.click(screen.getByRole('button', { name: 'Campaign / initiative status' }));
    const menu = await screen.findByRole('dialog', { name: 'Campaign / initiative status' });

    expect(menu).toHaveTextContent(
      'Approval progressNo activitiesPartly approvedAll approvedTimingNot startedRunningFinished',
    );
  });

  it('changes the year, and drops the plans of the old year', async () => {
    const initial = {
      ...standard2026(),
      filters: {
        ...standard2026().filters,
        plans: [{ id: 'P-1001', label: 'FY2026 Lubricants Europe' }],
      },
    };
    const { user } = renderBar({ initial });

    await user.click(screen.getByRole('button', { name: 'Year · 2026' }));
    const menu = await screen.findByRole('dialog', { name: 'Year' });
    await user.click(within(menu).getByRole('radio', { name: '2027' }));

    expect(current().year).toBe(2027);
    expect(current().filters.plans).toEqual([]);
    // The year is never a chip (FR-008).
    expect(screen.queryByRole('button', { name: /Remove filter: Year/ })).not.toBeInTheDocument();
  });

  it('chooses Strategic / AOP plans and Show levels, one value each', async () => {
    const { user } = renderBar();

    await user.click(screen.getByRole('button', { name: 'Strategic / AOP plans' }));
    await user.click(
      within(await screen.findByRole('dialog', { name: 'Strategic / AOP plans' })).getByRole(
        'radio',
        { name: 'Only' },
      ),
    );
    await user.keyboard('{Escape}');
    await user.click(screen.getByRole('button', { name: 'Show levels' }));
    await user.click(
      within(await screen.findByRole('dialog', { name: 'Show levels' })).getByRole('radio', {
        name: 'Initiatives without activities',
      }),
    );

    expect(current().filters).toMatchObject({
      strategic: 'only',
      showLevels: 'initiativesWithoutActivities',
    });
    expect(screen.getByRole('button', { name: 'Strategic / AOP plans · Only' })).toBeVisible();
    expect(
      screen.getByRole('button', {
        name: 'Remove filter: Show levels: Initiatives without activities',
      }),
    ).toBeVisible();
  });

  it('removes one chip, and Clear all clears every filter and the search but the year (FR-008)', async () => {
    const initial: ViewState = {
      ...standard2026(),
      year: 2027,
      search: 'helix',
      filters: {
        ...standard2026().filters,
        activityStatuses: ['declined', 'approved'],
        strategic: 'exclude',
      },
    };
    const { user } = renderBar({ initial });

    expect(screen.getByRole('searchbox', { name: 'Search' })).toHaveValue('helix');
    await user.click(
      screen.getByRole('button', { name: 'Remove filter: Activity status: Approved' }),
    );
    expect(current().filters.activityStatuses).toEqual(['declined']);

    await user.click(screen.getByRole('button', { name: 'Clear all' }));
    expect(current()).toMatchObject({ year: 2027, search: '', filters: standard2026().filters });
    expect(screen.getByRole('searchbox', { name: 'Search' })).toHaveValue('');
    expect(screen.queryByRole('button', { name: 'Clear all' })).not.toBeInTheDocument();
  });

  it('searches once typing stops (FR-006, FR-009)', async () => {
    const { user } = renderBar();

    await user.type(screen.getByRole('searchbox', { name: 'Search' }), 'A-1002');

    expect(current().search).toBe('');
    await waitFor(() => expect(current().search).toBe('A-1002'));
  });

  it.each([
    [{ shown: 3, total: 3, noun: 'activities' as const }, '3 of 3 activities'],
    [{ shown: 1, total: 40, noun: 'activities' as const }, '1 of 40 activities'],
    [{ shown: 1, total: 1, noun: 'activities' as const }, '1 of 1 activity'],
    [{ shown: 2, total: 5, noun: 'initiatives' as const }, '2 of 5 initiatives'],
    [{ shown: 0, total: 1240, noun: 'campaigns' as const }, '0 of 1,240 campaigns'],
  ])('counts what is shown: %o (FR-008)', (count, text) => {
    renderBar({ count });

    expect(screen.getByText(text)).toBeInTheDocument();
  });

  it('says "Updating…" next to the count while a new answer loads', () => {
    renderBar({ updating: true });

    expect(screen.getByText('3 of 3 activities')).toHaveTextContent('3 of 3 activities Updating…');
  });
});
