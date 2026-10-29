import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { expectNoAxeViolations } from '@/testing/axe';
import { allWorkOptions } from '@/testing/fixtures/all-work';
import { breakdown, BRANDS, group, SPEND_TYPE } from '@/testing/fixtures/all-work-breakdown';
import { testUser } from '@/testing/fixtures/users';
import { mockApi } from '@/testing/mock-api';
import type { MockedApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { apiError, reply } from '@/testing/replies';
import type { Answer } from '@/testing/replies';
import { PlannedCostByCard } from './PlannedCostByCard';

const PLAN = { id: 'P-1004', name: 'FY2027 Lubricants UK', year: 2027 };

const IN_GBP = breakdown({
  currency: 'GBP',
  groups: [
    group('SO-1301', 'Media', '23700.00', {
      share: '60.00',
      plannedCost: { amount: '23700.00', currency: 'GBP' },
    }),
    group('notOnLineItem', 'Not on a line item', '15800.00', {
      share: '40.00',
      plannedCost: { amount: '15800.00', currency: 'GBP' },
    }),
  ],
  total: group('total', 'Total', '39500.00', {
    share: '100.00',
    plannedCost: { amount: '39500.00', currency: 'GBP' },
  }),
});

function renderCard(answer: Answer<'GET /api/v1/all-work/breakdown'> = { body: IN_GBP }) {
  const api = mockApi([
    reply('GET /api/v1/all-work/options', {
      body: allWorkOptions({ year: 2027, attributes: [BRANDS, SPEND_TYPE] }),
    }),
    reply('GET /api/v1/all-work/breakdown', answer),
  ]);
  return { ...renderWithProviders(<PlannedCostByCard plan={PLAN} />, { user: testUser() }), api };
}

function lastQuery(api: MockedApi): URLSearchParams {
  const requests = api.requests.filter((r) => r.path === '/api/v1/all-work/breakdown');
  return new URLSearchParams(requests.at(-1)?.search);
}

describe('016 FR-042: Planned cost by on the plan page', () => {
  it('shows the plan’s planned cost by Spend type in the plan currency (US6 scenario 6)', async () => {
    const { api } = renderCard();

    const card = screen.getByRole('region', { name: 'Planned cost by' });
    const table = await within(card).findByRole('table');
    expect(within(table).getByRole('row', { name: /^Media/ })).toHaveTextContent('23,700.00 GBP');
    expect(within(table).getByRole('row', { name: /^Total/ })).toHaveTextContent('39,500.00 GBP');
    const query = lastQuery(api);
    expect({
      plan: query.getAll('plan'),
      year: query.get('year'),
      amountsIn: query.get('amountsIn'),
      by: query.get('by'),
    }).toEqual({ plan: ['P-1004'], year: '2027', amountsIn: 'plan', by: 'attr:SA-1030' });
    await expectNoAxeViolations();
  });

  it('breaks it down by another grouping chosen in the card', async () => {
    const { api, user } = renderCard();
    await screen.findByRole('table');

    await user.selectOptions(
      screen.getByRole('combobox', { name: 'Break down by' }),
      'attr:SA-1020',
    );

    await waitFor(() => expect(lastQuery(api).get('by')).toBe('attr:SA-1020'));
  });

  it('says when the plan has no planned cost yet', async () => {
    renderCard({ body: breakdown({ groups: [] }) });
    expect(await screen.findByText('No planned cost to break down yet.')).toBeVisible();
  });

  it('shows a failed load with Try again', async () => {
    let calls = 0;
    const { user } = renderCard(() => {
      calls += 1;
      return calls === 1
        ? { status: 500, body: apiError('internal_error', 'Something went wrong.') }
        : { body: IN_GBP };
    });

    await user.click(await screen.findByRole('button', { name: 'Try again' }));

    expect(await screen.findByRole('table')).toBeVisible();
  });

  it('keeps its place while the groups load', () => {
    renderCard(() => new Promise<never>(() => {}));
    expect(screen.getByRole('region', { name: 'Planned cost by' })).toBeVisible();
    expect(screen.queryByRole('table')).not.toBeInTheDocument();
  });
});
