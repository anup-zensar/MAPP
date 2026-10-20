import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { TreeTable } from '@/components';
import type { TreeNode } from '@/components';
import { formatDateRange, formatMoney } from '@/lib/format';
import type { Formatters } from '@/lib/useFormatters';
import { renderWithProviders } from '@/test/render';
import { campaignRows, CELL_KEYS, totalsFooter } from './campaignRows';
import type { RowOptions } from './campaignRows';
import type { PlanCampaigns } from './campaignApi';
import { budgetAndCost, campaign, initiative, planCampaigns, usd } from './fixtures';

// Feature 006 adds Add activity to initiative rows; here it stands in.
vi.mock('@/features/activities/activityTreeRows', () => ({
  useActivityRows: () => new Map(),
  activityActionsForInitiative: () => [{ label: 'Add activity', onSelect: () => {} }],
}));

const formatters = {
  dateRange: (start: string | null | undefined, end: string | null | undefined) =>
    formatDateRange(start, end),
  money: (value: { amount: string; currency: string } | null | undefined) =>
    value ? formatMoney(value.amount, value.currency) : '—',
} as Formatters;

function options(overrides: Partial<RowOptions> = {}): RowOptions {
  return { planId: 'P-1001', formatters, navigate: vi.fn(), ...overrides };
}

function allIds(nodes: TreeNode[]): string[] {
  return nodes.flatMap((node) => [node.id, ...allIds(node.children)]);
}

/** The rows as the plan tree shows them, every row expanded. */
function renderRows(data: PlanCampaigns, rowOptions: Partial<RowOptions> = {}) {
  const nodes = campaignRows(data, options(rowOptions));
  renderWithProviders(
    <TreeTable
      label="Plan tree"
      columns={CELL_KEYS.map((key) => ({ key, header: key }))}
      nodes={nodes}
      expandedIds={new Set(allIds(nodes))}
      onToggle={() => {}}
      footer={totalsFooter(data, formatters)}
    />,
  );
  return screen.getByRole('treegrid', { name: 'Plan tree' });
}

function rowOf(tree: HTMLElement, name: string): HTMLElement {
  return within(tree).getByRole('link', { name }).closest('tr') as HTMLElement;
}

const HELIX = campaign({
  budget: usd('120000.00'),
  outsidePlanYear: true,
  endDate: '2027-03-31',
  budgetAndCost: budgetAndCost({
    allocatedToChildren: usd('150000.00'),
    remainingForChildren: usd('-30000.00'),
    overAllocatedBy: usd('30000.00'),
    plannedCost: usd('150000.00'),
    budgetUsedPercent: 125,
    overBudget: true,
  }),
  counts: { initiatives: 2, activities: 5, approvedActivities: 3, lineItems: 0 },
  approvalProgress: { state: 'inProgress', approved: 3, total: 5 },
  initiatives: [
    initiative({
      owner: { id: 'U-1009', name: 'Sam Lee', initials: 'SL', jobTitle: '', active: false },
      counts: { activities: 1, approvedActivities: 0, lineItems: 0 },
    }),
    initiative({
      id: 'I-1002',
      name: 'Events',
      startDate: null,
      endDate: null,
      budget: null,
      timing: null,
    }),
  ],
});
const RIMULA = campaign({
  id: 'C-1002',
  name: 'Rimula fleet',
  startDate: '2026-03-01',
  endDate: '2026-12-31',
  budget: usd('200000.00'),
  budgetAndCost: budgetAndCost({ remainingForChildren: usd('200000.00') }),
  counts: { initiatives: 0, activities: 0, approvedActivities: 0, lineItems: 0 },
  initiatives: [],
});
const NO_BUDGET = campaign({
  id: 'C-1003',
  name: 'Sponsorships',
  startDate: null,
  endDate: null,
  budget: null,
  timing: null,
  counts: { initiatives: 0, activities: 0, approvedActivities: 0, lineItems: 0 },
  initiatives: [],
});
const DATA = planCampaigns([HELIX, RIMULA, NO_BUDGET], {
  campaigns: 3,
  initiatives: 2,
  activities: 1,
  allocatedToCampaigns: usd('320000.00'),
  plannedCost: usd('150000.00'),
});

describe('campaignRows (feature 005, US4)', () => {
  it('keeps the API’s order with initiatives under their campaign (FR-020)', () => {
    const nodes = campaignRows(DATA, options());

    expect(nodes.map((node) => node.id)).toEqual(['C-1001', 'C-1002', 'C-1003']);
    expect(nodes[0].children.map((node) => node.id)).toEqual(['I-1001', 'I-1002']);
    expect(nodes.map((node) => node.level)).toEqual(['campaign', 'campaign', 'campaign']);
    expect(nodes[0].children[0].level).toBe('initiative');
    expect(nodes[0].href).toBe('/plans/P-1001/campaigns/C-1001');
    expect(nodes[0].children[0].href).toBe('/plans/P-1001/initiatives/I-1001');
  });

  it('shows every cell of a campaign row (FR-019, FR-008, FR-011, FR-013)', () => {
    const tree = renderRows(DATA);
    const row = rowOf(tree, 'Helix launch');

    expect(row).toHaveTextContent('C-1001 · 2 initiatives');
    expect(within(row).getByText('Alex Morgan')).toBeInTheDocument();
    expect(row).toHaveTextContent('01 Jan 2026 – 31 Mar 2027');
    expect(
      within(row).getByRole('img', { name: "This campaign runs outside 2026, the plan's year" }),
    ).toBeInTheDocument();
    // At most two chips: the approval roll-up and one completeness chip (design system §4.1).
    expect(within(row).getByText('3 of 5 approved')).toBeInTheDocument();
    expect(within(row).getByText('3 of 5 activities approved')).toBeInTheDocument();
    expect(row).not.toHaveTextContent('In progress');
    expect(within(row).getByText('All filled in')).toBeInTheDocument();
    // The timing is a word under the dates, not a third chip.
    expect(within(row).getByText('Running')).toBeInTheDocument();
    expect(within(row).getByText('$120,000.00')).toBeInTheDocument();
    expect(within(row).getByText('Over-allocated by $30,000.00')).toBeInTheDocument();
    expect(within(row).getByText('$150,000.00')).toBeInTheDocument();
    // The budget used is the planned cost's second line (one column fewer, nothing clipped).
    expect(within(row).getByText('125% of budget').className).toMatch(/negative/);
    expect(
      within(row).getByRole('button', { name: 'Add initiative to Helix launch' }),
    ).toBeInTheDocument();
  });

  it('says what remains beneath a campaign budget, and nothing without one (FR-019)', () => {
    const tree = renderRows(DATA);

    const rimula = rowOf(tree, 'Rimula fleet');
    expect(rimula).toHaveTextContent('C-1002 · 0 initiatives');
    // One wording with the plan row: "Remaining …" (UI audit PL-03).
    expect(within(rimula).getByText('Remaining $200,000.00')).toBeInTheDocument();
    expect(rimula).not.toHaveTextContent('runs outside');

    const sponsorships = rowOf(tree, 'Sponsorships');
    expect(sponsorships).not.toHaveTextContent('Remaining');
    expect(sponsorships).not.toHaveTextContent('Over-allocated');
    // Without dates there is no timing word.
    expect(sponsorships).not.toHaveTextContent(/Running|Not started|Finished/);
  });

  it('shows every cell of an initiative row, inactive owners labelled (FR-019)', () => {
    const tree = renderRows(DATA);

    const digital = rowOf(tree, 'Digital & social');
    expect(digital).toHaveTextContent('I-1001 · 1 activity');
    expect(within(digital).getByText('Sam Lee (inactive)')).toBeInTheDocument();
    expect(digital).toHaveTextContent('01 Feb – 30 Nov 2026');
    expect(within(digital).getByText('$300,000.00')).toBeInTheDocument();
    expect(digital).not.toHaveTextContent('Remaining');

    const events = rowOf(tree, 'Events');
    expect(events).toHaveTextContent('I-1002 · 0 activities');
    expect(within(events).getAllByText('—').length).toBeGreaterThanOrEqual(3);
  });

  it('highlights the record just created (FR-004)', () => {
    const nodes = campaignRows(DATA, options({ highlightId: 'I-1002' }));

    expect(allIdsHighlighted(nodes)).toEqual(['I-1002']);
  });

  it('gives each level its ⋯ items, Delete last after a line (FR-021)', () => {
    const onDelete = vi.fn();
    const onCopy = vi.fn();
    const onMove = vi.fn();
    const navigate = vi.fn();
    const nodes = campaignRows(DATA, options({ onDelete, onCopy, onMove, navigate }));

    const campaignItems = nodes[0].actions ?? [];
    expect(campaignItems.map((item) => item.label)).toEqual([
      'Add initiative',
      'View',
      'Edit',
      'Copy',
      'Move to…',
      'Delete',
    ]);
    expect(campaignItems[5]).toMatchObject({ tone: 'danger', separatorBefore: true });
    const initiativeItems = nodes[0].children[0].actions ?? [];
    expect(initiativeItems.map((item) => item.label)).toEqual([
      'View',
      'Edit',
      'Copy',
      'Move to…',
      'Add activity',
      'Delete',
    ]);

    campaignItems[0].onSelect();
    expect(navigate).toHaveBeenLastCalledWith('/plans/P-1001/campaigns/C-1001/initiatives/new');
    initiativeItems[1].onSelect();
    expect(navigate).toHaveBeenLastCalledWith('/plans/P-1001/initiatives/I-1001/edit');
    initiativeItems[5].onSelect();
    expect(onDelete).toHaveBeenCalledWith(
      expect.objectContaining({ level: 'initiative', id: 'I-1001', name: 'Digital & social' }),
    );
  });

  it('opens Copy and Move to… with the record and the version the row showed (US6)', () => {
    const onCopy = vi.fn();
    const onMove = vi.fn();
    const nodes = campaignRows(DATA, options({ onCopy, onMove }));

    (nodes[0].actions ?? []).find((item) => item.label === 'Copy')?.onSelect();
    expect(onCopy).toHaveBeenCalledWith(
      expect.objectContaining({ level: 'campaign', id: 'C-1001', year: 2026, version: 3 }),
    );
    (nodes[0].children[0].actions ?? []).find((item) => item.label === 'Move to…')?.onSelect();
    expect(onMove).toHaveBeenCalledWith(
      expect.objectContaining({ level: 'initiative', id: 'I-1001', version: 3 }),
    );
  });

  it('lists Copy and Move to… only when they can be opened', () => {
    const nodes = campaignRows(DATA, options());

    expect((nodes[0].actions ?? []).map((item) => item.label)).toEqual([
      'Add initiative',
      'View',
      'Edit',
    ]);
  });

  it('shows the plan currency with the reader’s display currency beneath (016 FR-013, D-220)', () => {
    const euros = (amount: string) => ({ amount, currency: 'EUR' });
    const noGoals = { inq: null, mql: null, revenue: null };
    const helix = campaign({
      budget: usd('400000.00'),
      budgetAndCost: budgetAndCost({
        allocatedToChildren: usd('300000.00'),
        remainingForChildren: usd('100000.00'),
        plannedCost: usd('50000.00'),
        budgetUsedPercent: 13,
      }),
      display: {
        budget: euros('368000.00'),
        parentRemainingBudget: euros('184000.00'),
        budgetAndCost: budgetAndCost({
          allocatedToChildren: euros('276000.00'),
          remainingForChildren: euros('92000.00'),
          plannedCost: euros('46000.00'),
          budgetUsedPercent: 13,
        }),
        goals: noGoals,
      },
      initiatives: [
        initiative({
          budget: usd('300000.00'),
          display: {
            budget: euros('276000.00'),
            parentRemainingBudget: euros('92000.00'),
            budgetAndCost: budgetAndCost({ plannedCost: euros('46000.00') }),
            goals: noGoals,
          },
        }),
      ],
    });
    const tree = renderRows(
      planCampaigns(
        [helix],
        { allocatedToCampaigns: usd('400000.00'), plannedCost: usd('50000.00') },
        { allocatedToCampaigns: euros('368000.00'), plannedCost: euros('46000.00') },
      ),
    );

    const row = rowOf(tree, 'Helix launch');
    // Two currencies on the screen: both in the code form (design system §4.4).
    expect(row).toHaveTextContent('400,000.00 USD≈ about 368,000.00 EUR');
    expect(row).toHaveTextContent('Remaining 100,000.00 USD');
    expect(row).toHaveTextContent('50,000.00 USD≈ about 46,000.00 EUR');
    expect(row).not.toHaveTextContent('$');
    expect(rowOf(tree, 'Digital & social')).toHaveTextContent(
      '300,000.00 USD≈ about 276,000.00 EUR',
    );
    const totals = within(tree).getAllByRole('row').at(-1) as HTMLElement;
    expect(totals).toHaveTextContent('400,000.00 USD≈ about 368,000.00 EUR');
    expect(totals).not.toHaveTextContent('$');
  });

  it('sums the plan in the totals row, with singular forms (FR-022)', () => {
    const tree = renderRows(DATA);
    const totals = within(tree).getAllByRole('row').at(-1) as HTMLElement;

    // "Total" and its counts on two short lines, never wrapping (UI audit PL-05).
    expect(within(totals).getByText('Total')).toBeInTheDocument();
    expect(
      within(totals).getByText('3 campaigns · 2 initiatives · 1 activity'),
    ).toBeInTheDocument();
    expect(within(totals).getByText('$320,000.00')).toBeInTheDocument();
    expect(within(totals).getByText('Allocated')).toBeInTheDocument();
    expect(within(totals).getByText('$150,000.00')).toBeInTheDocument();

    const one = totalsFooter(
      planCampaigns([RIMULA], { campaigns: 1, initiatives: 1, activities: 0, plannedCost: null }),
      formatters,
    );
    renderWithProviders(<div data-testid="one">{one.name}</div>);
    expect(screen.getByTestId('one')).toHaveTextContent(
      'Total1 campaign · 1 initiative · 0 activities',
    );
    expect(one.plannedCost).toBe('—');
  });
});

function allIdsHighlighted(nodes: TreeNode[]): string[] {
  return nodes.flatMap((node) => [
    ...(node.highlighted ? [node.id] : []),
    ...allIdsHighlighted(node.children),
  ]);
}
