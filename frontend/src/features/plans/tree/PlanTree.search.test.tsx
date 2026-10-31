import { useState } from 'react';
import { screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { TreeNode } from '@/ui/TreeTable/TreeTable';
import { expectNoAxeViolations } from '@/testing/axe';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { apiError, reply } from '@/testing/replies';
import { testUser } from '@/testing/fixtures/users';
import { plan } from '@/testing/fixtures/plans';
import { PlanTree } from './PlanTree';
import { LINE_ITEMS_NOT_SEARCHED, usePlanSearchRows } from './usePlanSearchRows';

const SEARCH = 'GET /api/v1/plans/{planId}/search';
const VENUE_HIRE = {
  level: 'lineItem',
  id: 'L-1001',
  name: 'Venue hire',
  parentId: 'A-1001',
} as const;

const CAMPAIGNS: TreeNode[] = [
  {
    id: 'C-1001',
    level: 'campaign',
    name: 'Retail growth',
    cells: {},
    children: [
      {
        id: 'I-1001',
        level: 'initiative',
        name: 'Digital & social',
        cells: {},
        children: [
          { id: 'A-1001', level: 'activity', name: 'Social always-on', cells: {}, children: [] },
        ],
      },
    ],
  },
];

function Harness() {
  const [text, setText] = useState('');
  const found = usePlanSearchRows('P-1001', text);
  return (
    <PlanTree
      plan={plan()}
      childNodes={CAMPAIGNS}
      searchRows={found.rows}
      onSearch={setText}
      searchNote={found.note}
      searchPending={found.pending}
    />
  );
}

function renderSearch(answer: Parameters<typeof reply<typeof SEARCH>>[1]) {
  const api = mockApi([reply(SEARCH, answer, { planId: 'P-1001' })]);
  return {
    api,
    ...renderWithProviders(<Harness />, {
      user: testUser(),
      route: '/plans/P-1001',
      path: '/plans/:planId',
    }),
  };
}

function rowOf(text: string): HTMLElement | null {
  return screen.queryByText(text)?.closest<HTMLElement>('tr') ?? null;
}

const searchBox = () => screen.getByRole('searchbox', { name: 'Search name or ID in this plan' });

describe('039 US1: the plan tree search finds line items', () => {
  it('scenario 1: shows a matching line item under its activity, the path open, leading to its page', async () => {
    const { api, user } = renderSearch({ body: { items: [VENUE_HIRE] } });

    await user.type(searchBox(), 'venue');

    const link = await screen.findByRole('link', { name: 'Venue hire' });
    expect(link).toHaveAttribute('href', '/line-items/L-1001');
    const row = rowOf('Venue hire');
    expect(row).toHaveAttribute('aria-level', '5');
    expect(row).toHaveTextContent('LINE ITEM');
    expect(row).toHaveTextContent('L-1001');
    expect(rowOf('Social always-on')).toHaveAttribute('aria-expanded', 'true');
    expect(api.requests.find((r) => r.path === '/api/v1/plans/P-1001/search')?.search).toBe(
      '?q=venue',
    );
    await expectNoAxeViolations();
  });

  it('scenario 2: finds the line item by its ID', async () => {
    const { user } = renderSearch({ body: { items: [VENUE_HIRE] } });

    await user.type(searchBox(), 'l-1001');

    expect(await screen.findByRole('link', { name: 'Venue hire' })).toBeVisible();
  });

  it('scenarios 3 and 4: shows only matching line items, and none once the search is cleared', async () => {
    const { user } = renderSearch({ body: { items: [VENUE_HIRE] } });

    await user.type(searchBox(), 'venue');
    await screen.findByRole('link', { name: 'Venue hire' });
    await user.clear(searchBox());

    expect(rowOf('Venue hire')).toBeNull();
    expect(rowOf('Social always-on')).toBeInTheDocument();
  });

  it('scenario 5: says nothing matches once the search has answered', async () => {
    const { user } = renderSearch({ body: { items: [] } });

    await user.type(searchBox(), 'zzz');

    expect(await screen.findByText("Nothing in this plan matches 'zzz'")).toBeVisible();
  });

  it('keeps filtering the other rows and says so when the line items cannot be searched', async () => {
    const { user } = renderSearch({ status: 500, body: apiError('internal_error', 'Failed.') });

    await user.type(searchBox(), 'social');

    expect(await screen.findByRole('status')).toHaveTextContent(LINE_ITEMS_NOT_SEARCHED);
    await waitFor(() => expect(rowOf('Social always-on')).toBeInTheDocument());
    expect(rowOf('Retail growth')).toBeInTheDocument();
  });
});
