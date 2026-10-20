import { useState } from 'react';
import type { ReactNode } from 'react';
import {
  Button,
  ErrorState,
  filterTree,
  PersonLabel,
  SearchField,
  Skeleton,
  TreeTable,
} from '@/components';
import type { ActionsMenuItem, TreeNode, TreeTableColumn, TreeTableFooter } from '@/components';
import { BudgetUsed } from '@/features/campaigns/BudgetCost';
import { useFormatters } from '@/lib/useFormatters';
import { PlanCompleteness, PlanStatusChip } from '../PlanFacts';
import type { Plan } from '../plansApi';
import { NO_CAMPAIGNS, yearDates } from '../planText';
import styles from './PlanTree.module.css';

/**
 * The tree's columns (FR-013, design system §5.4): people never squeeze below 160 px and never
 * wrap, the dates and the two chips stay on one line, and money is right-aligned under its header.
 */
const COLUMNS: TreeTableColumn[] = [
  { key: 'name', header: 'Name', minWidth: '300px' },
  { key: 'owner', header: 'Owner', minWidth: '160px' },
  { key: 'dates', header: 'Dates', minWidth: '150px' },
  { key: 'status', header: 'Status' },
  { key: 'budget', header: 'Budget', align: 'end' },
  { key: 'plannedCost', header: 'Planned cost', align: 'end' },
  { key: 'budgetUsed', header: 'Budget used' },
];

/** Every row that has rows under it: what Expand all opens. */
function allIds(nodes: TreeNode[]): string[] {
  return nodes.flatMap((node) =>
    node.children.length > 0 ? [node.id, ...allIds(node.children)] : [],
  );
}

/**
 * The rows open before anyone expands or collapses them: the plan and campaign rows (005 FR-020)
 * and every row on the path to a highlighted one, the record just created, moved or copied.
 */
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
  /** The rows under the plan row: campaigns with their initiatives (feature 005). */
  childNodes?: TreeNode[];
  /** Buttons after Expand all / Collapse all, such as Add campaign (feature 005). */
  toolbarActions?: ReactNode;
  /** The totals row (feature 005). */
  footer?: TreeTableFooter;
  /** The plan row's ⋯ menu (feature 005: Add campaign). */
  planRowActions?: ActionsMenuItem[];
  /** Under the "No campaigns yet" sentence (feature 005: Add campaign). */
  emptyAction?: ReactNode;
  /** The rows below the plan are still loading (feature 005). */
  isLoading?: boolean;
  /** Reading the rows below the plan failed (feature 005); shown with Try again. */
  error?: unknown;
  /** Reads the rows below the plan again. */
  onRetry?: () => void;
}

/**
 * The plan tree (FR-013, FR-014): the plan row with its owner, dates, status, budget, planned cost
 * and budget used, and everything below it from later features; search on name or ID that keeps
 * the path to each match open, and Expand all / Collapse all.
 */
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
}: PlanTreeProps) {
  const { money, dateRange } = useFormatters();
  const [query, setQuery] = useState('');
  // What someone opened or closed, by row id; the other rows follow openAtFirst, so rows that load
  // later (campaigns) start the way 005 FR-020 says.
  const [toggled, setToggled] = useState<ReadonlyMap<string, boolean>>(() => new Map());
  // After Expand all or Collapse all, every row not toggled since follows that choice, also rows whose
  // children load afterwards (activities arrive in a query of their own, after the campaigns).
  const [everyRow, setEveryRow] = useState<boolean | null>(null);
  const { figures } = plan;

  const planNode: TreeNode = {
    id: plan.id,
    level: 'plan',
    name: plan.name,
    actions: planRowActions,
    children: childNodes,
    cells: {
      name: <span className={styles.id}>{plan.id}</span>,
      // Like every other row: avatar and name (UI audit PL-04).
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
        <span className={styles.money}>
          {money(figures.budget)}
          {figures.overAllocatedBy ? (
            <span className={styles.problem}>
              Over-allocated by {money(figures.overAllocatedBy)}
            </span>
          ) : (
            <span className={styles.muted}>Remaining {money(figures.remainingBudget)}</span>
          )}
        </span>
      ) : (
        <span className={styles.muted}>No budget set</span>
      ),
      plannedCost: money(figures.plannedCost),
      budgetUsed: (
        <BudgetUsed
          percent={figures.budgetUsedPercent}
          overBudget={figures.overBudget}
          label={`Budget used of ${plan.name}`}
          compact
        />
      ),
    },
  };

  const filtered = filterTree([planNode], query);
  const openFirst = new Set(openAtFirst([planNode]));
  const expandedIds = new Set(
    allIds([planNode]).filter((id) => toggled.get(id) ?? everyRow ?? openFirst.has(id)),
  );

  /** Sets every row in ids open or closed. */
  function setOpen(ids: Iterable<string>, open: boolean) {
    setToggled((current) => {
      const next = new Map(current);
      for (const id of ids) {
        next.set(id, open);
      }
      return next;
    });
  }

  /** Opens or closes every row, now and when more rows load (Expand all, Collapse all). */
  function setAllOpen(open: boolean) {
    setToggled(new Map());
    setEveryRow(open);
  }

  /** Filters the rows and opens the path to every match. */
  function search(text: string) {
    setQuery(text);
    const { openIds } = filterTree([planNode], text);
    if (openIds.size > 0) {
      setOpen(openIds, true);
    }
  }

  return (
    // A table card (design system §5.3): the toolbar and the table reach the card's edges.
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
      <div className={styles.tree}>
        <TreeTable
          label="Plan tree"
          columns={COLUMNS}
          nodes={filtered.nodes}
          expandedIds={expandedIds}
          onToggle={(id) => setOpen([id], !expandedIds.has(id))}
          emptyText={`Nothing in this plan matches '${query.trim()}'`}
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
