/**
 * Which rows of All work are open and on screen, and how they are paged (data-model §4, FR-011,
 * FR-012, D-115). The web app never adds money: these functions only choose which of the answered
 * rows are on screen (D-114).
 */
import type { AllWorkRow, ViewState } from './all-work-types';

/** One row on screen, with whether it is open and how many rows it holds. */
export interface VisibleRow {
  row: AllWorkRow;
  expanded: boolean;
  /** The rows directly under it that can be shown (unassigned rows only while switched on). */
  childCount: number;
}

/** The keys of the rows that have a search hit somewhere below them (data-model §4). */
export function rowsWithHitBelow(rows: AllWorkRow[]): Set<string> {
  const parentOf = new Map(rows.map((row) => [row.key, row.parentKey]));
  const keys = new Set<string>();
  for (const row of rows) {
    if (!row.searchHit) continue;
    let parent = row.parentKey;
    while (parent && !keys.has(parent)) {
      keys.add(parent);
      parent = parentOf.get(parent) ?? null;
    }
  }
  return keys;
}

/**
 * Whether a row is open (data-model §4, FR-011): by default plans, campaigns and initiatives are
 * open and activities closed; with a search, only the rows with a hit below them are open (a
 * matching container shows its contents, collapsed); Expand all and Collapse all open or close
 * everything. A row that was opened or closed by hand is the other way round.
 */
export function isExpanded(
  row: AllWorkRow,
  state: ViewState,
  hitBelow: Set<string>,
  toggled: Set<string> = new Set(state.expansion.toggled),
): boolean {
  let open: boolean;
  switch (state.expansion.mode) {
    case 'allExpanded':
      open = true;
      break;
    case 'allCollapsed':
      open = false;
      break;
    default:
      open =
        state.search.trim() !== ''
          ? hitBelow.has(row.key)
          : row.level === 'plan' || row.level === 'campaign' || row.level === 'initiative';
  }
  return open !== toggled.has(row.key);
}

/**
 * The rows on screen, in order (data-model §4): every row whose parents are all open, without the
 * unassigned rows while the switch is off (FR-012).
 */
export function visibleRows(rows: AllWorkRow[], state: ViewState): VisibleRow[] {
  const shown = state.showUnassigned ? rows : rows.filter((row) => row.level !== 'unassigned');
  const childCount = new Map<string, number>();
  for (const row of shown) {
    if (row.parentKey) {
      childCount.set(row.parentKey, (childCount.get(row.parentKey) ?? 0) + 1);
    }
  }
  const hitBelow = rowsWithHitBelow(rows);
  const toggled = new Set(state.expansion.toggled);
  const openKeys = new Set<string>();
  const result: VisibleRow[] = [];
  for (const row of shown) {
    if (row.parentKey && !openKeys.has(row.parentKey)) {
      continue;
    }
    const count = childCount.get(row.key) ?? 0;
    const expanded = count > 0 && isExpanded(row, state, hitBelow, toggled);
    if (expanded) {
      openKeys.add(row.key);
    }
    result.push({ row, expanded, childCount: count });
  }
  return result;
}

/** Opens a closed row or closes an open one (FR-011). */
export function toggleRow(state: ViewState, key: string): ViewState {
  const toggled = state.expansion.toggled.includes(key)
    ? state.expansion.toggled.filter((other) => other !== key)
    : [...state.expansion.toggled, key];
  return { ...state, expansion: { ...state.expansion, toggled } };
}

/** One page of the rows on screen (D-115). */
export interface RowPage {
  rows: VisibleRow[];
  /** The page shown, 1 or more (a page beyond the last becomes the last). */
  page: number;
  pageCount: number;
  /** The position of the first and last row of the page, from 1; 0 when there are no rows. */
  from: number;
  to: number;
  total: number;
}

export function pageOf(rows: VisibleRow[], page: number, pageSize: number): RowPage {
  const pageCount = Math.max(1, Math.ceil(rows.length / pageSize));
  const current = Math.min(Math.max(1, page), pageCount);
  const start = (current - 1) * pageSize;
  const pageRows = rows.slice(start, start + pageSize);
  return {
    rows: pageRows,
    page: current,
    pageCount,
    from: pageRows.length > 0 ? start + 1 : 0,
    to: start + pageRows.length,
    total: rows.length,
  };
}

/**
 * "FY2026 Lubricants Europe › Helix launch": the rows above the first row of a page that are on
 * an earlier page, for "Continued: <path>" (D-115). Empty when the page starts with a top row.
 */
export function continuedPath(page: VisibleRow[], all: AllWorkRow[]): string {
  const first = page[0]?.row;
  if (!first?.parentKey) {
    return '';
  }
  const byKey = new Map(all.map((row) => [row.key, row]));
  const names: string[] = [];
  let parent = byKey.get(first.parentKey);
  while (parent) {
    names.unshift(parent.name);
    parent = parent.parentKey ? byKey.get(parent.parentKey) : undefined;
  }
  return names.join(' › ');
}
