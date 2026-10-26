import { useState } from 'react';
import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { TreeNode } from '@/ui/TreeTable/TreeTable';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/users';
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
    expect(planRow).toHaveTextContent('01 Jan – 31 Dec 2026');
    expect(planRow).toHaveTextContent('Draft');
    expect(planRow).toHaveTextContent('$1,200,000.00Remaining $1,200,000.00');
    expect(planRow).toHaveTextContent('0%');
    expect(
      screen.getByText('No campaigns yet — a campaign is a theme that runs for most of the year.'),
    ).toBeInTheDocument();
  });

  it("writes the plan row's dates in the person's date format, like every other row (D-216, 003 FR-024)", () => {
    const user = testUser();
    renderWithProviders(<PlanTree plan={plan()} childNodes={[]} />, {
      user: { ...user, preferences: { ...user.preferences, dateFormat: 'DD/MM/YYYY' } },
      route: '/plans/P-1001',
      path: '/plans/:planId',
    });

    const planRow = within(screen.getByRole('treegrid', { name: 'Plan tree' })).getAllByRole(
      'row',
    )[1];
    expect(planRow).toHaveTextContent('01/01/2026 – 31/12/2026');
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

  it('starts with the plan and campaign rows open, and expands and collapses everything', async () => {
    const { user } = renderTree(CAMPAIGNS);

    expect(screen.getAllByRole('row')).toHaveLength(5);
    expect(screen.getByText('Digital & social')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Collapse Retail growth' }));
    expect(screen.queryByText('Digital & social')).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Expand all' }));
    expect(screen.getByText('Digital & social')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Collapse all' }));
    expect(screen.getAllByRole('row')).toHaveLength(2);
    await user.click(screen.getByRole('button', { name: 'Expand FY2026 Lubricants Europe' }));
    expect(screen.getByText('Retail growth')).toBeInTheDocument();
  });

  it('keeps rows open that get rows below them after Expand all (activities load after campaigns)', async () => {
    const [retail, fleet] = CAMPAIGNS;
    const withActivity: TreeNode[] = [
      {
        ...retail,
        children: [
          {
            ...retail.children[0],
            children: [
              { id: 'A-1001', level: 'activity', name: 'Launch film', cells: {}, children: [] },
            ],
          },
        ],
      },
      fleet,
    ];
    function LateActivities() {
      const [loaded, setLoaded] = useState(false);
      return (
        <>
          <button type="button" onClick={() => setLoaded(true)}>
            Load activities
          </button>
          <PlanTree plan={plan()} childNodes={loaded ? withActivity : CAMPAIGNS} />
        </>
      );
    }
    const { user } = renderWithProviders(<LateActivities />, {
      user: testUser(),
      route: '/plans/P-1001',
      path: '/plans/:planId',
    });

    await user.click(screen.getByRole('button', { name: 'Expand all' }));
    await user.click(screen.getByRole('button', { name: 'Load activities' }));
    expect(screen.getByText('Launch film')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Collapse Digital & social' })).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Collapse all' }));
    expect(screen.getAllByRole('row')).toHaveLength(2);
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

  it('opens the path to the highlighted row (005 FR-004)', () => {
    const [retail, fleet] = CAMPAIGNS;
    renderTree([
      {
        ...retail,
        children: [
          {
            ...retail.children[0],
            children: [
              {
                id: 'A-1001',
                level: 'activity',
                name: 'Launch film',
                cells: {},
                children: [],
                highlighted: true,
              },
            ],
          },
        ],
      },
      fleet,
    ]);

    expect(screen.getByText('Launch film')).toBeInTheDocument();
  });

  it('shows Try again when the rows below the plan cannot be read', () => {
    renderWithProviders(<PlanTree plan={plan()} error={new Error('down')} onRetry={() => {}} />, {
      user: testUser(),
    });

    expect(screen.getByRole('button', { name: 'Try again' })).toBeInTheDocument();
    expect(
      screen.queryByText(
        'No campaigns yet — a campaign is a theme that runs for most of the year.',
      ),
    ).not.toBeInTheDocument();
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
