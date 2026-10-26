import { screen, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { closestTo } from '@/testing/dom';
import { gridResult } from '@/testing/fixtures/all-work';
import { renderAllWorkPage, rowNames, viewQueries } from '@/testing/fixtures/all-work-page';
import { reply } from '@/testing/replies';
import { resetAllWorkStore } from './all-work-store';
import type { AllWorkResult } from './all-work-types';

afterEach(() => {
  resetAllWorkStore();
  vi.restoreAllMocks();
});

describe('AllWorkPage filters and search (US2)', () => {
  /** The answer to "Activity status: Declined": one branch, its parents as context. */
  function declinedOnly(): AllWorkResult {
    const keep = ['P-1001', 'C-1001', 'I-1001', 'A-1002', 'L-1002', 'A-1002:unassigned'];
    return gridResult({
      rows: gridResult()
        .rows.filter((candidate) => keep.includes(candidate.key))
        .map((candidate) =>
          ['P-1001', 'C-1001', 'I-1001'].includes(candidate.key)
            ? { ...candidate, role: 'context' as const }
            : candidate,
        ),
      summary: { ...gridResult().summary, campaigns: 0, initiatives: 0, activities: 1 },
      count: { shown: 1, total: 3, noun: 'activities' },
    });
  }

  it('applies a filter at once: the rows, the chip, the count and the cards follow (US2 scenario 2, FR-004, FR-005)', async () => {
    const { api, user } = renderAllWorkPage([
      reply('GET /api/v1/all-work', (request) =>
        new URL(request.url).searchParams.get('activityStatus') === 'declined'
          ? { body: declinedOnly() }
          : { body: gridResult() },
      ),
    ]);
    expect(await screen.findByText('3 of 3 activities')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Activity status' }));
    await user.click(
      within(await screen.findByRole('dialog', { name: 'Activity status' })).getByRole('checkbox', {
        name: 'Declined',
      }),
    );
    await user.keyboard('{Escape}');

    expect(await screen.findByText('1 of 3 activities')).toBeInTheDocument();
    expect(viewQueries(api).at(-1)?.getAll('activityStatus')).toEqual(['declined']);
    expect(
      screen.getByRole('button', { name: 'Remove filter: Activity status: Declined' }),
    ).toBeInTheDocument();
    expect(rowNames()).toEqual([
      'FY2026 Lubricants Europe',
      'Helix launch',
      'Helix digital',
      'Helix social ads',
      'Total · 3 activities',
    ]);
    expect(screen.getAllByText('Shown for context')).toHaveLength(3);
    const summary = screen.getByRole('region', { name: 'Summary' });
    expect(within(summary).getByText('Campaigns').parentElement).toHaveTextContent('0');
  });

  it('finds an activity by its ID: its path opens and the text is marked (US2 scenario 1, FR-006, SC-001)', async () => {
    const found = gridResult({
      rows: declinedOnly().rows.map((candidate) =>
        candidate.key === 'A-1002'
          ? { ...candidate, role: 'match' as const, searchHit: { fields: ['id' as const] } }
          : candidate,
      ),
      count: { shown: 1, total: 3, noun: 'activities' },
    });
    const { api, user } = renderAllWorkPage([
      reply('GET /api/v1/all-work', (request) =>
        new URL(request.url).searchParams.get('search') === 'A-1002'
          ? { body: found }
          : { body: gridResult() },
      ),
    ]);
    await screen.findByText('3 of 3 activities');

    await user.type(screen.getByRole('searchbox', { name: 'Search' }), 'A-1002');

    expect(await screen.findByText('1 of 3 activities')).toBeInTheDocument();
    expect(viewQueries(api).map((query) => query.get('search'))).not.toContain('A');
    // The path to the hit is open; the activity itself stays closed.
    expect(rowNames()).toEqual([
      'FY2026 Lubricants Europe',
      'Helix launch',
      'Helix digital',
      'Helix social ads',
      'Total · 3 activities',
    ]);
    const activity = closestTo(screen.getByRole('rowheader', { name: /Helix social ads/ }), 'tr');
    expect([...activity.querySelectorAll('mark')].map((mark) => mark.textContent)).toEqual([
      'A-1002',
    ]);
  });

  it('keeps the filters and the search while the person moves around MAPP (FR-009)', async () => {
    const { api, user, router } = renderAllWorkPage([
      reply('GET /api/v1/all-work', (request) =>
        new URL(request.url).searchParams.has('activityStatus')
          ? { body: declinedOnly() }
          : { body: gridResult() },
      ),
    ]);
    await screen.findByText('3 of 3 activities');
    await user.click(screen.getByRole('button', { name: 'Activity status' }));
    await user.click(
      within(await screen.findByRole('dialog')).getByRole('checkbox', { name: 'Declined' }),
    );
    await screen.findByText('1 of 3 activities');

    await router.navigate('/plans');
    await screen.findByText('Another page');
    await router.navigate('/all-work');

    expect(await screen.findByText('1 of 3 activities')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Remove filter: Activity status: Declined' }),
    ).toBeInTheDocument();
    expect(viewQueries(api)).toHaveLength(2);
  });

  it('counts initiatives under "Initiatives without activities" (US2 scenario 4, FR-007, FR-008)', async () => {
    const result = gridResult({
      rows: gridResult().rows.filter((candidate) => ['P-1002', 'C-1003'].includes(candidate.key)),
      count: { shown: 1, total: 2, noun: 'initiatives' },
    });
    renderAllWorkPage(
      [reply('GET /api/v1/all-work', { body: result })],
      '/all-work?showLevels=initiativesWithoutActivities',
    );

    expect(await screen.findByText('1 of 2 initiatives')).toBeInTheDocument();
    expect(
      screen.getByRole('button', {
        name: 'Remove filter: Show levels: Initiatives without activities',
      }),
    ).toBeInTheDocument();
  });
});
