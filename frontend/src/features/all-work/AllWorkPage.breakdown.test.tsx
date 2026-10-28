import { screen, waitFor, within } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { allWorkOptions } from '@/testing/fixtures/all-work';
import {
  breakdown,
  BRANDS,
  COUNTRY,
  GROUPINGS,
  group,
  SPEND_TYPE,
} from '@/testing/fixtures/all-work-breakdown';
import { renderAllWorkPage, viewQueries } from '@/testing/fixtures/all-work-page';
import { fieldHelpReply } from '@/testing/fixtures/field-help-texts';
import { expectNoAxeViolations } from '@/testing/axe';
import type { MockedApi } from '@/testing/mock-api';
import { apiError, reply } from '@/testing/replies';
import type { Answer } from '@/testing/replies';
import { resetAllWorkStore } from './all-work-store';

afterEach(() => resetAllWorkStore());

const OPTIONS = reply('GET /api/v1/all-work/options', {
  body: allWorkOptions({ attributes: [COUNTRY, BRANDS, SPEND_TYPE] }),
});

const BY_COUNTRY = breakdown({
  by: GROUPINGS[2] ?? GROUPINGS[0],
  thenBy: GROUPINGS[1] ?? null,
  groups: [
    group('SO-1101', 'Germany', '6000.00', {
      share: '60.00',
      budget: { amount: '5000.00', currency: 'USD' },
      budgetUsedPercent: 120,
      overBudget: true,
      children: [group('AT-1001', 'Event', '6000.00', { share: '60.00', children: undefined })],
    }),
    group('SO-1102', 'France', '4000.00', { share: '40.00' }),
  ],
  countsInEach: true,
  notes: ['Activities with several values count in each group; the total counts them once.'],
});

function breakdownQueries(api: MockedApi): URLSearchParams[] {
  return api.requests
    .filter((request) => request.path === '/api/v1/all-work/breakdown')
    .map((request) => new URLSearchParams(request.search));
}

async function openBreakdown(answer: Answer<'GET /api/v1/all-work/breakdown'>) {
  const page = renderAllWorkPage([OPTIONS, reply('GET /api/v1/all-work/breakdown', answer)]);
  await screen.findByRole('link', { name: 'Helix launch' });
  await page.user.click(screen.getByRole('tab', { name: 'Breakdown' }));
  return page;
}

describe('016 FR-039: More filters', () => {
  it('filters by Brands › Helix and Not filled in, shown as one chip (US6 scenario 1)', async () => {
    const { api, user } = renderAllWorkPage([OPTIONS]);
    await screen.findByRole('link', { name: 'Helix launch' });

    await user.click(screen.getByRole('button', { name: 'More filters' }));
    const menu = await screen.findByRole('dialog', { name: 'More filters' });
    const brands = within(menu).getByRole('group', { name: 'Brands' });
    await user.click(within(brands).getByRole('checkbox', { name: 'Helix' }));
    await user.click(within(brands).getByRole('checkbox', { name: 'Not filled in' }));

    expect(
      screen.getByRole('button', { name: 'Remove filter: Brands: Helix, Not filled in' }),
    ).toBeVisible();
    await waitFor(() =>
      expect(viewQueries(api).at(-1)?.getAll('attr')).toEqual(['SA-1020:SO-1201,none']),
    );
    expect(screen.getByRole('button', { name: 'More filters · 1' })).toBeVisible();
    await expectNoAxeViolations();
  });

  it('finds attributes and values by search, under their level', async () => {
    const { user } = renderAllWorkPage([OPTIONS]);
    await screen.findByRole('link', { name: 'Helix launch' });
    await user.click(screen.getByRole('button', { name: 'More filters' }));
    const menu = await screen.findByRole('dialog', { name: 'More filters' });
    expect(within(menu).getByText('Campaign')).toBeVisible();

    await user.type(within(menu).getByRole('searchbox'), 'rimu');

    expect(within(menu).queryByRole('group', { name: 'Country / OU' })).not.toBeInTheDocument();
    const brands = within(menu).getByRole('group', { name: 'Brands' });
    expect(within(menu).getAllByRole('checkbox')).toEqual([
      within(brands).getByRole('checkbox', { name: 'Rimula' }),
    ]);
  });
});

describe('016 FR-040, FR-041: the Breakdown view', () => {
  it('breaks the filtered planned cost down by Spend type, with Not on a line item and the totals (US6 scenario 4)', async () => {
    const { api } = await openBreakdown({ body: breakdown() });

    const table = await screen.findByRole('table', { name: 'Breakdown' });
    const rows = within(table)
      .getAllByRole('row')
      .map((row) => row.textContent);
    expect(rows).toEqual([
      'Spend typeActivitiesLine itemsPlanned costShare of planned cost',
      'Media11$6,000.0060.00 %',
      'Not on a line item10$4,000.0040.00 %',
      'Total21$10,000.00100.00 %',
    ]);
    expect(breakdownQueries(api).at(-1)?.get('by')).toBe('attr:SA-1030');
    expect(screen.getByRole('combobox', { name: 'Break down by' })).toHaveValue('attr:SA-1030');
    await expectNoAxeViolations();
  });

  it('opens each group into its then-by groups, with budgets and the note on several values (US6 scenarios 3, 5)', async () => {
    const { api, user } = await openBreakdown({ body: BY_COUNTRY });
    await screen.findByRole('table', { name: 'Breakdown' });

    await user.selectOptions(screen.getByRole('combobox', { name: 'then by' }), 'activityType');
    await waitFor(() => expect(breakdownQueries(api).at(-1)?.get('thenBy')).toBe('activityType'));
    await user.click(screen.getByRole('button', { name: "Show the groups of 'Germany'" }));

    expect(screen.getByRole('rowheader', { name: 'Event' })).toBeVisible();
    expect(screen.getByRole('cell', { name: '120 %' })).toBeVisible();
    expect(
      screen.getByText(
        'Activities with several values count in each group; the total counts them once.',
      ),
    ).toBeVisible();
    await expectNoAxeViolations();
  });

  it('opens a group in the Grid with its filter (FR-040)', async () => {
    const { api, user } = await openBreakdown({ body: BY_COUNTRY });

    await user.click(await screen.findByRole('button', { name: 'Germany' }));

    expect(screen.getByRole('tab', { name: 'Grid' })).toHaveAttribute('aria-selected', 'true');
    expect(
      screen.getByRole('button', { name: 'Remove filter: Country / OU: Germany' }),
    ).toBeVisible();
    await waitFor(() =>
      expect(viewQueries(api).at(-1)?.getAll('attr')).toEqual(['SA-1013:SO-1101']),
    );
  });

  it('explains the Share column with its Field help text (016 FR-047)', async () => {
    const page = renderAllWorkPage([
      OPTIONS,
      reply('GET /api/v1/all-work/breakdown', { body: breakdown() }),
      fieldHelpReply({ 'allWork.breakdownShare': 'Share of the total planned cost.' }),
    ]);
    await screen.findByRole('link', { name: 'Helix launch' });
    await page.user.click(screen.getByRole('tab', { name: 'Breakdown' }));

    expect(
      await screen.findByRole('button', { name: 'About Share of planned cost' }),
    ).toBeVisible();
  });

  it('says when nothing has a planned cost', async () => {
    await openBreakdown({ body: breakdown({ groups: [] }) });
    expect(await screen.findByText('No planned cost to break down yet.')).toBeVisible();
  });

  it('shows a failed load with Try again, which asks again', async () => {
    let calls = 0;
    const { api, user } = await openBreakdown(() => {
      calls += 1;
      return calls === 1
        ? { status: 500, body: apiError('internal_error', 'Something went wrong.') }
        : { body: breakdown() };
    });

    await user.click(await screen.findByRole('button', { name: 'Try again' }));

    expect(await screen.findByRole('table', { name: 'Breakdown' })).toBeVisible();
    expect(breakdownQueries(api)).toHaveLength(2);
  });

  it('shows the skeleton while the groups load', async () => {
    await openBreakdown(() => new Promise<never>(() => {}));
    expect(screen.queryByRole('table', { name: 'Breakdown' })).not.toBeInTheDocument();
    expect(screen.queryByRole('combobox', { name: 'Break down by' })).not.toBeInTheDocument();
  });
});
