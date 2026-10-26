import { describe, expect, it } from 'vitest';
import { nth } from '@/testing/dom';
import { seedRows } from '@/testing/fixtures/all-work-seed';
import { standard2026 } from '@/testing/fixtures/all-work';
import type { AllWorkRow, ViewState } from './all-work-types';
import { continuedPath, pageOf, toggleRow, visibleRows } from './view-rows';

function keysOf(state: ViewState, rows: AllWorkRow[] = seedRows()) {
  return visibleRows(rows, state).map((visible) => visible.row.key);
}

describe('visibleRows (FR-011, FR-012, FR-006)', () => {
  it('opens plans, campaigns and initiatives and closes activities by default', () => {
    expect(keysOf(standard2026())).toEqual([
      'P-1001',
      'C-1001',
      'I-1001',
      'A-1001',
      'A-1002',
      'C-1002',
      'I-1002',
      'A-1003',
      'P-1002',
      'C-1003',
    ]);
    const [plan] = visibleRows(seedRows(), standard2026());
    expect(plan).toMatchObject({ expanded: true, childCount: 2 });
  });

  it('opens or closes one row against the mode', () => {
    const opened = toggleRow(standard2026(), 'A-1001');
    expect(keysOf(opened)).toContain('L-1001');
    expect(keysOf(opened)).toContain('A-1001:unassigned');

    const closedAgain = toggleRow(opened, 'A-1001');
    expect(keysOf(closedAgain)).not.toContain('L-1001');

    const planClosed = toggleRow(standard2026(), 'P-1001');
    expect(keysOf(planClosed)).toEqual(['P-1001', 'P-1002', 'C-1003']);
  });

  it('opens everything with Expand all and nothing with Collapse all', () => {
    const all = { ...standard2026(), expansion: { mode: 'allExpanded' as const, toggled: [] } };
    expect(keysOf(all)).toHaveLength(seedRows().length);

    const none = { ...standard2026(), expansion: { mode: 'allCollapsed' as const, toggled: [] } };
    expect(keysOf(none)).toEqual(['P-1001', 'P-1002']);
  });

  it('hides the unassigned rows while the switch is off, and counts them only when shown (FR-012)', () => {
    const all = { ...standard2026(), expansion: { mode: 'allExpanded' as const, toggled: [] } };
    expect(keysOf({ ...all, showUnassigned: false })).not.toContain('A-1001:unassigned');
    const activity = visibleRows(seedRows(), { ...all, showUnassigned: false }).find(
      (visible) => visible.row.key === 'A-1001',
    );
    expect(activity?.childCount).toBe(1);
  });

  it('opens the path to a search hit and keeps everything else closed (FR-006)', () => {
    const rows = seedRows().map((row) =>
      row.key === 'L-1002' ? { ...row, searchHit: { fields: ['name' as const] } } : row,
    );
    const state = { ...standard2026(), search: 'paid' };

    expect(keysOf(state, rows)).toEqual([
      'P-1001',
      'C-1001',
      'I-1001',
      'A-1001',
      'A-1002',
      'L-1002',
      'A-1002:unassigned',
      'C-1002',
      'P-1002',
    ]);
  });
});

describe('pageOf and continuedPath (D-115)', () => {
  const all = { ...standard2026(), expansion: { mode: 'allExpanded' as const, toggled: [] } };
  const rows = visibleRows(seedRows(), all);

  it('cuts the rows into pages and keeps the page within the last one', () => {
    expect(pageOf(rows, 1, 5)).toMatchObject({ page: 1, pageCount: 3, from: 1, to: 5, total: 14 });
    expect(pageOf(rows, 9, 5)).toMatchObject({ page: 3, from: 11, to: 14 });
    expect(pageOf([], 1, 50)).toMatchObject({ page: 1, pageCount: 1, from: 0, to: 0, total: 0 });
  });

  it('names the rows above a page that starts inside a branch', () => {
    const second = pageOf(rows, 2, 5).rows;
    expect(nth(second, 0).row.key).toBe('A-1001:unassigned');
    expect(continuedPath(second, seedRows())).toBe(
      'FY2026 Lubricants Europe › Helix launch › Helix digital › Helix launch event',
    );
    expect(continuedPath(pageOf(rows, 1, 5).rows, seedRows())).toBe('');
  });
});
