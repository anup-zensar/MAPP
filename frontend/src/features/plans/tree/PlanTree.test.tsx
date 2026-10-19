import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { TreeNode } from '@/components';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { plan } from '../testData';
import { PlanTree } from './PlanTree';

/** Two campaigns, one with an initiative: the rows feature 005 adds under the plan. */
const CAMPAIGNS: TreeNode[] = [
  {
    id: 'C-1001',
    level: 'campaign',
    name: 'Retail growth',
    cells: {},
    children: [
      { id: 'I-1001', level: 'initiative', name: 'Digital & social', cells: {}, children: [] },
    ],
  },
  { id: 'C-1002', level: 'campaign', name: 'Fleet deals', cells: {}, children: [] },
];

function renderTree(children: TreeNode[] = []) {
  return renderWithProviders(<PlanTree plan={plan()} childNodes={children} />, {
    user: testUser(),
    route: '/plans/P-1001',
    path: '/plans/:planId',
  });
}

describe('PlanTree (FR-013, FR-014)', () => {
  it('shows the plan row with its facts and the empty text', () => {
    renderTree();

    const tree = screen.getByRole('treegrid', { name: 'Plan tree' });
    const planRow = within(tree).getAllByRole('row')[1];
    expect(planRow).toHaveAttribute('aria-level', '1');
    expect(planRow).toHaveTextContent('PLAN');
    expect(planRow).toHaveTextContent('FY2026 Lubricants Europe');
    expect(planRow).toHaveTextContent('P-1001');
    expect(planRow).toHaveTextContent('Priya Raman');
    expect(planRow).toHaveTextContent('1 Jan – 31 Dec 2026');
    expect(planRow).toHaveTextContent('Draft');
    expect(planRow).toHaveTextContent('$1,200,000.00Remaining $1,200,000.00');
    expect(planRow).toHaveTextContent('0 %');
    expect(
      screen.getByText('No campaigns yet — a campaign is a theme that runs for most of the year.'),
    ).toBeInTheDocument();
  });

  it('says how far the campaigns go over the plan budget', () => {
    renderWithProviders(
      <PlanTree
        plan={plan({
          figures: {
            ...plan().figures,
            remainingBudget: { amount: '-100000.00', currency: 'USD' },
            overAllocatedBy: { amount: '100000.00', currency: 'USD' },
          },
        })}
      />,
      { user: testUser() },
    );

    expect(screen.getByText('Over-allocated by $100,000.00')).toBeInTheDocument();
  });

  it('says "No budget set" without a plan budget', () => {
    renderWithProviders(
      <PlanTree
        plan={plan({
          figures: {
            ...plan().figures,
            budget: null,
            remainingBudget: null,
            budgetUsedPercent: null,
          },
        })}
      />,
      { user: testUser() },
    );

    expect(screen.getByText('No budget set')).toBeInTheDocument();
  });

  it('expands and collapses everything', async () => {
    const { user } = renderTree(CAMPAIGNS);

    expect(screen.getAllByRole('row')).toHaveLength(4);
    await user.click(screen.getByRole('button', { name: 'Expand all' }));
    expect(screen.getByText('Digital & social')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Collapse all' }));
    expect(screen.getAllByRole('row')).toHaveLength(2);
    await user.click(screen.getByRole('button', { name: 'Expand FY2026 Lubricants Europe' }));
    expect(screen.getByText('Retail growth')).toBeInTheDocument();
  });

  it('searches name or ID, keeping the path to each match open', async () => {
    const { user } = renderTree(CAMPAIGNS);

    await user.type(
      screen.getByRole('searchbox', { name: 'Search name or ID in this plan' }),
      'i-1001',
    );

    expect(screen.getByText('Digital & social')).toBeInTheDocument();
    expect(screen.getByText('Retail growth')).toBeInTheDocument();
    expect(screen.queryByText('Fleet deals')).not.toBeInTheDocument();
  });

  it('says when nothing matches', async () => {
    const { user } = renderTree(CAMPAIGNS);

    await user.type(
      screen.getByRole('searchbox', { name: 'Search name or ID in this plan' }),
      'zzz',
    );

    expect(screen.getByText("Nothing in this plan matches 'zzz'")).toBeInTheDocument();
  });
});
