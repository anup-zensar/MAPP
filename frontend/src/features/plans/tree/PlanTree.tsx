import { useState } from 'react';
import type { ReactNode } from 'react';
import { Button } from '@/ui/Button/Button';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { PersonLabel } from '@/ui/PersonLabel/PersonLabel';
import { SearchField } from '@/ui/SearchField/SearchField';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { TreeTable } from '@/ui/TreeTable/TreeTable';
import { filterTree } from '@/ui/TreeTable/filter-tree';
import type { ActionsMenuItem } from '@/ui/ActionsMenu/ActionsMenu';
import type { TreeNode, TreeTableColumn, TreeTableFooter } from '@/ui/TreeTable/TreeTable';
import { useFormatters } from '@/format/useFormatters';
import { BudgetCell, PlannedCostCell } from '../campaigns/BudgetCells';
import { PlanCompleteness, PlanStatusChip } from '../PlanFacts';
import type { Plan } from '../api';
import { NO_CAMPAIGNS, yearDates } from '../plan-text';
import styles from './PlanTree.module.css';

// The name takes what the other columns leave; people and dates never squeeze, so the tree fits 1,440 px.
const COLUMNS: TreeTableColumn[] = [
  { key: 'name', header: 'Name' },
  { key: 'owner', header: 'Owner', minWidth: '160px' },
  { key: 'dates', header: 'Dates', minWidth: '164px' },
  { key: 'status', header: 'Status' },
  { key: 'budget', header: 'Budget', align: 'end' },
  { key: 'plannedCost', header: 'Planned cost', align: 'end' },
];

/** Every row that has rows under it: what Expand all opens. */
function allIds(nodes: TreeNode[]): string[] {
  return nodes.flatMap((node) =>
    node.children.length > 0 ? [node.id, ...allIds(node.children)] : [],
  );
}

/** The plan and campaign rows, and the path to the record just created, moved or copied. */
function openAtFirst(nodes: TreeNode[], path: string[] = []): string[] {
  return nodes.flatMap((node) => {
    const below = openAtFirst(node.children, [...path, node.id]);
    const own = node.level === 'plan' || node.level === 'campaign' ? [node.id] : [];
    const toHighlight = node.highlighted ? path : [];
    return [...own, ...toHighlight, ...below];
  });
}

export interface PlanTreeProps {
  plan: Plan;
  childNodes?: TreeNode[];
  toolbarActions?: ReactNode;
  footer?: TreeTableFooter;
  planRowActions?: ActionsMenuItem[];
  emptyAction?: ReactNode;
  isLoading?: boolean;
  error?: unknown;
  onRetry?: () => void;
  /** The line items that match the search, by activity ID: shown under their activity while searching. */
  searchRows?: ReadonlyMap<string, TreeNode[]>;
  onSearch?: (text: string) => void;
  searchNote?: string | null;
  searchPending?: boolean;
}

const SEARCHING = 'Searching this plan…';
const NO_ROWS: ReadonlyMap<string, TreeNode[]> = new Map();

function withRowsBelow(nodes: TreeNode[], rows: ReadonlyMap<string, TreeNode[]>): TreeNode[] {
  return nodes.map((node) => ({
    ...node,
    children: [...withRowsBelow(node.children, rows), ...(rows.get(node.id) ?? [])],
  }));
}

export function PlanTree({
  plan,
  childNodes = [],
  toolbarActions,
  footer,
  planRowActions,
  emptyAction,
  isLoading = false,
  error,
  onRetry,
  searchRows = NO_ROWS,
  onSearch,
  searchNote,
  searchPending = false,
}: PlanTreeProps) {
  const { dateRange } = useFormatters();
  const [query, setQuery] = useState('');
  // What someone opened or closed, by row id; the other rows follow openAtFirst, also rows loaded later.
  const [toggled, setToggled] = useState<ReadonlyMap<string, boolean>>(() => new Map());
  const [everyRow, setEveryRow] = useState<boolean | null>(null);
  const { figures } = plan;
  const planFigures = plan.planFigures ?? figures;

  const planNode: TreeNode = {
    id: plan.id,
    level: 'plan',
    name: plan.name,
    actions: planRowActions,
    children: query.trim() ? withRowsBelow(childNodes, searchRows) : childNodes,
    cells: {
      name: <span className={styles.id}>{plan.id}</span>,
      owner: plan.owner ? (
        <span className={styles.nowrap}>
          <PersonLabel person={plan.owner} />
        </span>
      ) : (
        <span className={styles.muted}>No owner yet</span>
      ),
      dates: <span className={styles.nowrap}>{yearDates(plan.year, dateRange)}</span>,
      status: (
        <span className={styles.chips}>
          <PlanStatusChip status={plan.status} />
          <PlanCompleteness planId={plan.id} completeness={plan.completeness} compact />
        </span>
      ),
      budget: figures.budget ? (
        <BudgetCell
          budget={planFigures.budget}
          displayBudget={figures.budget}
          remaining={planFigures.remainingBudget}
        />
      ) : (
        <span className={styles.muted}>No budget set</span>
      ),
      plannedCost: (
        <PlannedCostCell
          cost={planFigures.plannedCost}
          displayCost={figures.plannedCost}
          percent={figures.budgetUsedPercent}
          overBudget={figures.overBudget}
        />
      ),
    },
  };

  const filtered = filterTree([planNode], query);
  const openFirst = new Set(openAtFirst([planNode]));
  const searchOpen = query.trim() ? filtered.openIds : new Set<string>();
  const expandedIds = new Set(
    allIds([planNode]).filter(
      (id) => toggled.get(id) ?? (searchOpen.has(id) || (everyRow ?? openFirst.has(id))),
    ),
  );

  function setOpen(ids: Iterable<string>, open: boolean) {
    setToggled((current) => {
      const next = new Map(current);
      for (const id of ids) {
        next.set(id, open);
      }
      return next;
    });
  }

  function setAllOpen(open: boolean) {
    setToggled(new Map());
    setEveryRow(open);
  }

  function search(text: string) {
    setQuery(text);
    onSearch?.(text);
    // Paths a search opened stay open, also those to line items found by the API after the last key.
    const opened = new Set([...filterTree([planNode], text).openIds, ...searchOpen]);
    if (opened.size > 0) {
      setOpen(opened, true);
    }
  }

  return (
    <section className={styles.card} aria-label="Plan tree">
      <div className={styles.toolbar}>
        <div className={styles.search}>
          <SearchField
            label="Search name or ID in this plan"
            placeholder="Search name or ID in this plan"
            value={query}
            onChange={search}
          />
        </div>
        <div className={styles.buttons}>
          <Button onClick={() => setAllOpen(true)}>Expand all</Button>
          <Button onClick={() => setAllOpen(false)}>Collapse all</Button>
          {toolbarActions}
        </div>
      </div>
      {searchNote && query.trim() && (
        <div className={styles.empty} role="status">
          <p>{searchNote}</p>
        </div>
      )}
      <div className={styles.tree}>
        <TreeTable
          label="Plan tree"
          columns={COLUMNS}
          nodes={filtered.nodes}
          expandedIds={expandedIds}
          onToggle={(id) => setOpen([id], !expandedIds.has(id))}
          emptyText={searchPending ? SEARCHING : `Nothing in this plan matches '${query.trim()}'`}
          footer={footer}
        />
        {error ? (
          <div className={styles.state}>
            <ErrorState error={error} onRetry={() => onRetry?.()} />
          </div>
        ) : isLoading ? (
          <div className={styles.state} aria-busy="true" aria-label="Loading the campaigns">
            <Skeleton lines={3} />
          </div>
        ) : (
          childNodes.length === 0 &&
          !query.trim() && (
            <div className={styles.empty}>
              <p>{NO_CAMPAIGNS}</p>
              {emptyAction}
            </div>
          )
        )}
      </div>
    </section>
  );
}
