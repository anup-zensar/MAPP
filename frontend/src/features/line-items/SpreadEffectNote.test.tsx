import { screen, waitFor } from '@testing-library/react';
import type { QueryClient } from '@tanstack/react-query';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/testing/mockApi';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/users';
import { SpreadEffectNote } from './SpreadEffectNote';
import { eur, launchFilmSection, lineItem, spreadMonth, usd } from './testData';

const SECTION = 'GET /api/v1/activities/A-1001/line-items';

function custom(id: string, months: string[]) {
  return lineItem({
    id,
    name: id,
    spread: {
      method: 'custom',
      state: 'spread',
      months: months.map((month) => spreadMonth(month, eur('100.00'), usd('108.70'))),
      customTotal: eur('100.00'),
      addsUp: false,
    },
  });
}

function renderNote(
  dates: { startDate: string; endDate: string; ongoing: boolean },
  items = [
    lineItem({ id: 'L-1' }),
    lineItem({ id: 'L-2', spread: { ...lineItem().spread, method: 'firstMonth' } }),
    custom('L-3', ['2026-03', '2026-04']),
  ],
) {
  mockApi({ [SECTION]: { body: launchFilmSection({ items }) } });
  return renderWithProviders(<SpreadEffectNote activityId="A-1001" {...dates} />, {
    user: testUser(),
  });
}

/** Waits until the line items have arrived. */
async function loaded(queryClient: QueryClient) {
  await waitFor(() =>
    expect(queryClient.getQueryData(['line-items', 'activity', 'A-1001'])).toBeDefined(),
  );
}

describe('SpreadEffectNote (FR-022, D-150)', () => {
  it('says how many spreads follow the new dates and which custom months fall outside', async () => {
    renderNote({ startDate: '2026-04-01', endDate: '2026-06-30', ongoing: false });

    expect(await screen.findByRole('status')).toHaveTextContent(
      "When you save, 2 line items are re-spread over the new dates and 1 custom spread is kept as typed. 1 custom spread will have months outside the activity's dates.",
    );
  });

  it('says nothing while the dates are the saved ones', async () => {
    const { queryClient } = renderNote({
      startDate: '2026-03-01',
      endDate: '2026-05-31',
      ongoing: false,
    });

    await loaded(queryClient);
    expect(screen.queryByRole('status')).not.toBeInTheDocument();
  });

  it('says nothing without line items', async () => {
    const { queryClient } = renderNote(
      { startDate: '2026-04-01', endDate: '2026-06-30', ongoing: false },
      [],
    );

    await loaded(queryClient);
    expect(screen.queryByRole('status')).not.toBeInTheDocument();
  });

  it('says that line items are not spread without dates, in the singular', async () => {
    renderNote({ startDate: '', endDate: '', ongoing: false }, [
      lineItem({ id: 'L-1' }),
      custom('L-3', ['2026-03']),
    ]);

    expect(await screen.findByRole('status')).toHaveTextContent(
      "When you save, 1 custom spread is kept as typed. Without dates, 1 line item won't be spread until the activity has dates again.",
    );
  });

  it('counts an ongoing activity as dates', async () => {
    renderNote({ startDate: '2026-03-01', endDate: '', ongoing: true }, [lineItem({ id: 'L-1' })]);

    expect(await screen.findByRole('status')).toHaveTextContent(
      'When you save, 1 line item is re-spread over the new dates.',
    );
  });
});
