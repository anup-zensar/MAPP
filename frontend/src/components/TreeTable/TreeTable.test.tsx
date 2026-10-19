import { screen, waitFor, within } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { TreeTable } from './TreeTable';
import type { TreeNode } from './TreeTable';

const COLUMNS = [
  { key: 'name', header: 'Name' },
  { key: 'owner', header: 'Owner' },
  { key: 'budget', header: 'Budget', align: 'end' as const },
];

function tree(onDelete = vi.fn()): TreeNode[] {
  return [
    {
      id: 'P-1001',
      level: 'plan',
      name: 'FY2026 Lubricants Europe',
      cells: { owner: 'Priya Raman', budget: '$1,200,000.00' },
      children: [
        {
          id: 'C-1001',
          level: 'campaign',
          name: 'Helix launch',
          href: '/plans/P-1001/campaigns/C-1001',
          cells: { name: 'C-1001 · 2 initiatives', owner: 'Alex Morgan', budget: '$500,000.00' },
          actions: [{ label: 'Delete', tone: 'danger', onSelect: onDelete }],
          children: [
            {
              id: 'I-1001',
              level: 'initiative',
              name: 'Digital & social',
              href: '/plans/P-1001/initiatives/I-1001',
              cells: {},
              children: [],
            },
            {
              id: 'I-1002',
              level: 'initiative',
              name: 'Retail displays',
              cells: {},
              children: [],
              highlighted: true,
            },
          ],
        },
        { id: 'C-1002', level: 'campaign', name: 'Fleet cards', cells: {}, children: [] },
      ],
    },
  ];
}

function Example({ nodes = tree(), start = ['P-1001'] }: { nodes?: TreeNode[]; start?: string[] }) {
  const [expanded, setExpanded] = useState(new Set(start));
  return (
    <TreeTable
      label="Plan tree"
      columns={COLUMNS}
      nodes={nodes}
      expandedIds={expanded}
      onToggle={(id) =>
        setExpanded((current) => {
          const next = new Set(current);
          if (next.has(id)) {
            next.delete(id);
          } else {
            next.add(id);
          }
          return next;
        })
      }
      emptyText="No campaigns yet — a campaign is a theme that runs for most of the year."
      footer={{ name: 'Total · 2 campaigns', budget: '$500,000.00' }}
    />
  );
}

function rowNames() {
  const grid = screen.getByRole('treegrid', { name: 'Plan tree' });
  return within(grid)
    .getAllByRole('row')
    .slice(1, -1)
    .map((row) => row.getAttribute('data-row-id'));
}

describe('TreeTable', () => {
  it('is a treegrid of the expanded rows with levels, cells and a totals row (design system §5)', () => {
    renderWithProviders(<Example />);

    expect(rowNames()).toEqual(['P-1001', 'C-1001', 'C-1002']);
    const plan = screen.getByText('FY2026 Lubricants Europe').closest('tr');
    expect(plan).toHaveAttribute('aria-level', '1');
    expect(plan).toHaveAttribute('aria-expanded', 'true');
    expect(plan).toHaveAttribute('tabindex', '0');
    const campaign = screen.getByText('Helix launch').closest('tr');
    expect(campaign).toHaveAttribute('aria-level', '2');
    expect(campaign).toHaveAttribute('aria-expanded', 'false');
    expect(campaign).toHaveAttribute('tabindex', '-1');
    expect(within(campaign as HTMLElement).getByText('CAMPAIGN')).toBeInTheDocument();
    expect(within(campaign as HTMLElement).getByText('C-1001 · 2 initiatives')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Helix launch' })).toHaveAttribute(
      'href',
      '/plans/P-1001/campaigns/C-1001',
    );
    expect(screen.getByText('Total · 2 campaigns')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Actions for Helix launch' })).toBeInTheDocument();
  });

  it('expands and collapses with the chevron buttons', async () => {
    const { user } = renderWithProviders(<Example />);

    await user.click(screen.getByRole('button', { name: 'Expand Helix launch' }));
    expect(rowNames()).toEqual(['P-1001', 'C-1001', 'I-1001', 'I-1002', 'C-1002']);

    await user.click(screen.getByRole('button', { name: 'Collapse FY2026 Lubricants Europe' }));
    expect(rowNames()).toEqual(['P-1001']);
  });

  it('moves with the arrow keys, Home and End, and opens and closes rows (treegrid)', async () => {
    const { user } = renderWithProviders(<Example />);
    const row = (id: string) =>
      screen.getAllByRole('row').find((element) => element.getAttribute('data-row-id') === id);

    await user.tab();
    expect(row('P-1001')).toHaveFocus();

    await user.keyboard('{ArrowDown}');
    expect(row('C-1001')).toHaveFocus();
    expect(row('C-1001')).toHaveAttribute('tabindex', '0');

    await user.keyboard('{ArrowRight}');
    expect(row('C-1001')).toHaveAttribute('aria-expanded', 'true');
    await user.keyboard('{ArrowRight}');
    expect(row('I-1001')).toHaveFocus();

    await user.keyboard('{ArrowLeft}');
    expect(row('C-1001')).toHaveFocus();
    await user.keyboard('{ArrowLeft}');
    expect(row('C-1001')).toHaveAttribute('aria-expanded', 'false');

    await user.keyboard('{End}');
    expect(row('C-1002')).toHaveFocus();
    await user.keyboard('{Home}');
    expect(row('P-1001')).toHaveFocus();
  });

  it('opens the record with Enter', async () => {
    const { user, router } = renderWithProviders(<Example />);

    await user.tab();
    await user.keyboard('{ArrowDown}{Enter}');

    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/plans/P-1001/campaigns/C-1001'),
    );
  });

  it('runs the row actions from its ⋯ menu', async () => {
    const onDelete = vi.fn();
    const { user } = renderWithProviders(<Example nodes={tree(onDelete)} />);

    await user.click(screen.getByRole('button', { name: 'Actions for Helix launch' }));
    await user.click(screen.getByRole('menuitem', { name: 'Delete' }));

    await waitFor(() => expect(onDelete).toHaveBeenCalledOnce());
  });

  it('says so when there is nothing in the tree', () => {
    renderWithProviders(<Example nodes={[]} />);

    expect(
      screen.getByText('No campaigns yet — a campaign is a theme that runs for most of the year.'),
    ).toBeInTheDocument();
  });

  it('tints the highlighted row', async () => {
    const { user } = renderWithProviders(<Example />);

    await user.click(screen.getByRole('button', { name: 'Expand Helix launch' }));

    expect(screen.getByText('Retail displays').closest('tr')?.className).toMatch(/highlighted/);
  });
});
