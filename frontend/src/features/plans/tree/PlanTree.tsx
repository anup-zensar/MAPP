import { useState } from 'react';
import type { ReactNode } from 'react';
import { Button, Card, filterTree, ProgressBar, SearchField, TreeTable } from '@/components';
import type { ActionsMenuItem, TreeNode, TreeTableColumn, TreeTableFooter } from '@/components';
import { useFormatters } from '@/lib/useFormatters';
import { PlanCompleteness, PlanStatusChip } from '../PlanFacts';
import type { Plan } from '../plansApi';
import { NO_CAMPAIGNS, yearDates } from '../planText';
import styles from './PlanTree.module.css';

const COLUMNS: TreeTableColumn[] = [
  { key: 'name', header: 'Name' },
  { key: 'owner', header: 'Owner' },
  { key: 'dates', header: 'Dates' },
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
}: PlanTreeProps) {
  const { money } = useFormatters();
  const [query, setQuery] = useState('');
  const [expandedIds, setExpandedIds] = useState(() => new Set([plan.id]));
  const { figures } = plan;

  const planNode: TreeNode = {
    id: plan.id,
    level: 'plan',
    name: plan.name,
    actions: planRowActions,
    children: childNodes,
    cells: {
      name: <span className={styles.id}>{plan.id}</span>,
      owner: plan.owner ? plan.owner.name : <span className={styles.muted}>No owner yet</span>,
      dates: yearDates(plan.year),
      status: (
        <span className={styles.chips}>
          <PlanStatusChip status={plan.status} />
          <PlanCompleteness planId={plan.id} completeness={plan.completeness} />
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
      budgetUsed:
        figures.budgetUsedPercent === null ? (
          '—'
        ) : (
          <span className={styles.used}>
            <ProgressBar
              value={figures.budgetUsedPercent}
              label={`Budget used of ${plan.name}`}
              tone={figures.overBudget ? 'problem' : undefined}
            />
            <span className={figures.overBudget ? styles.problem : undefined}>
              {figures.budgetUsedPercent} %
            </span>
          </span>
        ),
    },
  };

  const filtered = filterTree([planNode], query);

  /** Filters the rows and opens the path to every match. */
  function search(text: string) {
    setQuery(text);
    const { openIds } = filterTree([planNode], text);
    if (openIds.size > 0) {
      setExpandedIds((current) => new Set([...current, ...openIds]));
    }
  }

  function toggle(id: string) {
    setExpandedIds((current) => {
      const next = new Set(current);
      if (!next.delete(id)) {
        next.add(id);
      }
      return next;
    });
  }

  return (
    <Card title="Plan tree" subtitle="The plan and everything in it">
      <div className={styles.tree}>
        <div className={styles.toolbar}>
          <div className={styles.search}>
            <SearchField
              label="Search name or ID in this plan"
              placeholder="Search name or ID in this plan"
              value={query}
              onChange={search}
            />
          </div>
          <Button size="sm" onClick={() => setExpandedIds(new Set(allIds([planNode])))}>
            Expand all
          </Button>
          <Button size="sm" onClick={() => setExpandedIds(new Set())}>
            Collapse all
          </Button>
          {toolbarActions}
        </div>
        <TreeTable
          label="Plan tree"
          columns={COLUMNS}
          nodes={filtered.nodes}
          expandedIds={expandedIds}
          onToggle={toggle}
          emptyText={`Nothing in this plan matches '${query.trim()}'`}
          footer={footer}
        />
        {childNodes.length === 0 && !query.trim() && (
          <div className={styles.empty}>
            <p>{NO_CAMPAIGNS}</p>
            {emptyAction}
          </div>
        )}
      </div>
    </Card>
  );
}
