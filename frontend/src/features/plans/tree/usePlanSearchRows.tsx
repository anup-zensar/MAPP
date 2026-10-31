import { useMemo } from 'react';
import { useDebounced } from '@/features/activities';
import type { TreeNode } from '@/ui/TreeTable/TreeTable';
import { usePlanSearch } from '../api';
import type { PlanSearchMatch } from '../api';
import styles from './PlanTree.module.css';

const SEARCH_DELAY_MS = 300;
export const LINE_ITEMS_NOT_SEARCHED = "Line items couldn't be searched. Try again.";

export interface PlanSearchRows {
  rows: ReadonlyMap<string, TreeNode[]>;
  note: string | null;
  pending: boolean;
}

export function lineItemRows(matches: PlanSearchMatch[]): ReadonlyMap<string, TreeNode[]> {
  const rows = new Map<string, TreeNode[]>();
  for (const match of matches) {
    if (match.level !== 'lineItem') {
      continue;
    }
    const row: TreeNode = {
      id: match.id,
      level: 'lineItem',
      name: match.name,
      href: `/line-items/${encodeURIComponent(match.id)}`,
      cells: { name: <span className={styles.id}>{match.id}</span> },
      children: [],
    };
    rows.set(match.parentId, [...(rows.get(match.parentId) ?? []), row]);
  }
  return rows;
}

/** The plan's line items that match `text`, asked of the API once typing pauses. */
export function usePlanSearchRows(planId: string, text: string): PlanSearchRows {
  const settled = useDebounced(text.trim(), SEARCH_DELAY_MS);
  const search = usePlanSearch(planId, settled);
  const items = search.data?.items;
  const rows = useMemo(() => lineItemRows(items ?? []), [items]);
  const typing = settled !== text.trim();
  return {
    rows,
    note: search.isError ? LINE_ITEMS_NOT_SEARCHED : null,
    pending: text.trim() !== '' && (typing || search.isFetching),
  };
}
