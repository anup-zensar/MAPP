import { within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { TreeNode } from '@/ui/TreeTable/TreeTable';
import { nth } from '@/testing/dom';
import { DATA, renderRows, rowOf, rowOptions } from '@/testing/fixtures/campaign-rows';
import { campaignRows } from './campaign-rows';
import type { RowOptions } from './campaign-rows';

function options(overrides: Partial<RowOptions> = {}): RowOptions {
  return rowOptions(vi.fn(), overrides);
}

describe('campaignRows (feature 005, US4)', () => {
  it('keeps the API’s order with initiatives under their campaign (FR-020)', () => {
    const nodes = campaignRows(DATA, options());

    expect(nodes.map((node) => node.id)).toEqual(['C-1001', 'C-1002', 'C-1003']);
    const helix = nth(nodes, 0);
    expect(helix.children.map((node) => node.id)).toEqual(['I-1001', 'I-1002']);
    expect(nodes.map((node) => node.level)).toEqual(['campaign', 'campaign', 'campaign']);
    expect(nth(helix.children, 0).level).toBe('initiative');
    expect(helix.href).toBe('/plans/P-1001/campaigns/C-1001');
    expect(nth(helix.children, 0).href).toBe('/plans/P-1001/initiatives/I-1001');
  });

  it('shows every cell of a campaign row (FR-019, FR-008, FR-011, FR-013)', () => {
    const tree = renderRows(DATA, options());
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
    // On screen the short form keeps the Budget column narrow; screen readers hear the whole word.
    expect(within(row).getByText('Over by $30,000.00')).toHaveAttribute('aria-hidden', 'true');
    expect(within(row).getByText('Over-allocated by $30,000.00')).toHaveClass('visually-hidden');
    expect(within(row).getByText('$150,000.00')).toBeInTheDocument();
    // The budget used is the planned cost's second line (one column fewer, nothing clipped).
    expect(within(row).getByText('125% used').className).toMatch(/negative/);
    expect(
      within(row).getByRole('button', { name: 'Add initiative to Helix launch' }),
    ).toBeInTheDocument();
  });

  it('says what remains beneath a campaign budget, and nothing without one (FR-019)', () => {
    const tree = renderRows(DATA, options());

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
    const tree = renderRows(DATA, options());

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

    const campaignItems = nth(nodes, 0).actions ?? [];
    expect(campaignItems.map((item) => item.label)).toEqual([
      'Add initiative',
      'View',
      'Edit',
      'Copy',
      'Move to…',
      'Delete',
    ]);
    expect(nth(campaignItems, 5)).toMatchObject({ tone: 'danger', separatorBefore: true });
    const initiativeItems = nth(nth(nodes, 0).children, 0).actions ?? [];
    expect(initiativeItems.map((item) => item.label)).toEqual([
      'View',
      'Edit',
      'Copy',
      'Move to…',
      'Add activity',
      'Delete',
    ]);

    nth(campaignItems, 0).onSelect();
    expect(navigate).toHaveBeenLastCalledWith('/plans/P-1001/campaigns/C-1001/initiatives/new');
    nth(initiativeItems, 1).onSelect();
    expect(navigate).toHaveBeenLastCalledWith('/plans/P-1001/initiatives/I-1001/edit');
    nth(initiativeItems, 5).onSelect();
    expect(onDelete).toHaveBeenCalledWith(
      expect.objectContaining({ level: 'initiative', id: 'I-1001', name: 'Digital & social' }),
    );
  });

  it('opens Copy and Move to… with the record and the version the row showed (US6)', () => {
    const onCopy = vi.fn();
    const onMove = vi.fn();
    const nodes = campaignRows(DATA, options({ onCopy, onMove }));

    (nth(nodes, 0).actions ?? []).find((item) => item.label === 'Copy')?.onSelect();
    expect(onCopy).toHaveBeenCalledWith(
      expect.objectContaining({ level: 'campaign', id: 'C-1001', year: 2026, version: 3 }),
    );
    (nth(nth(nodes, 0).children, 0).actions ?? [])
      .find((item) => item.label === 'Move to…')
      ?.onSelect();
    expect(onMove).toHaveBeenCalledWith(
      expect.objectContaining({ level: 'initiative', id: 'I-1001', version: 3 }),
    );
  });

  it('lists Copy and Move to… only when they can be opened', () => {
    const nodes = campaignRows(DATA, options());

    expect((nth(nodes, 0).actions ?? []).map((item) => item.label)).toEqual([
      'Add initiative',
      'View',
      'Edit',
    ]);
  });
});

function allIdsHighlighted(nodes: TreeNode[]): string[] {
  return nodes.flatMap((node) => [
    ...(node.highlighted ? [node.id] : []),
    ...allIdsHighlighted(node.children),
  ]);
}
