import { Plus } from 'lucide-react';
import type { NavigateFunction } from 'react-router';
import { Button, CompletenessChip, PersonLabel, Tooltip } from '@/components';
import type { ActionsMenuItem, TreeNode, TreeTableFooter } from '@/components';
import { fieldHref, recordHref } from '@/lib/recordLinks';
import type { Formatters } from '@/lib/useFormatters';
import { BudgetUsed, CampaignBudgetCell } from './BudgetCost';
import styles from './campaignRows.module.css';
import type { Campaign, Initiative, PlanCampaigns } from './campaignApi';
import { initiativeExtraActions } from './extensions';
import { ApprovalProgressChip, OutsideYearNote, TimingChip } from './StatusChips';
import {
  activitiesCount,
  ADD_INITIATIVE,
  addInitiativeTo,
  ALLOCATED_TO_CAMPAIGNS,
  DELETE,
  EDIT,
  initiativesCount,
  totalsText,
  VIEW,
} from './texts';
import type { RecordLevel } from './texts';

/**
 * The plan tree's cells, by column key (feature 004's PlanTree columns Name, Owner, Dates, Status,
 * Budget, Planned cost, Budget used): `name` (shown after the name: ID, child count and the quick
 * action), `owner`, `dates`, `status`, `budget`, `plannedCost`, `budgetUsed`.
 */
export const CELL_KEYS = [
  'name',
  'owner',
  'dates',
  'status',
  'budget',
  'plannedCost',
  'budgetUsed',
] as const;

/** A campaign or initiative a row action is about. */
export interface RecordTarget {
  level: RecordLevel;
  id: string;
  name: string;
  planId: string;
  /** What goes with it (the delete dialog). */
  counts: {
    initiatives?: number;
    activities: number;
    approvedActivities: number;
    lineItems: number;
  };
}

export interface RowOptions {
  planId: string;
  formatters: Formatters;
  navigate: NavigateFunction;
  /** The record just created, moved or copied (location state `highlight`, FR-004). */
  highlightId?: string;
  /** Feature 006's activity rows under each initiative (extensions.ts). */
  rowsBelow?: Map<string, TreeNode[]>;
  /** **Delete** (US5): opens the delete dialog; the item is listed only with it. */
  onDelete?: (target: RecordTarget) => void;
}

/** The record a row stands for, for its actions. */
export function targetOf(record: Campaign | Initiative): RecordTarget {
  return 'initiatives' in record
    ? {
        level: 'campaign',
        id: record.id,
        name: record.name,
        planId: record.planId,
        counts: record.counts,
      }
    : {
        level: 'initiative',
        id: record.id,
        name: record.name,
        planId: record.planId,
        counts: record.counts,
      };
}

/**
 * The plan tree's rows of a plan's campaigns (FR-019 – FR-021): one node per campaign in the
 * API's order (FR-020) with its initiatives as children, each with its level, link, ID, child
 * count, owner, dates, chips, money, budget used and ⋯ actions.
 */
export function campaignRows(data: PlanCampaigns, options: RowOptions): TreeNode[] {
  return data.items.map((item) => ({
    id: item.id,
    level: 'campaign',
    name: item.name,
    href: recordHref('campaign', item.id, options.planId),
    highlighted: item.id === options.highlightId,
    cells: {
      ...commonCells(item, options),
      name: (
        <span className={styles.nameExtra}>
          <span>
            {item.id} · {initiativesCount(item.counts.initiatives)}
          </span>
          <Tooltip content={addInitiativeTo(item.name)}>
            <Button
              variant="ghost"
              size="sm"
              icon={<Plus />}
              aria-label={addInitiativeTo(item.name)}
              className={styles.quickAction}
              onClick={() => void options.navigate(addInitiativeHref(options.planId, item.id))}
            />
          </Tooltip>
        </span>
      ),
      dates: (
        <span className={styles.nowrap}>
          {options.formatters.dateRange(item.startDate, item.endDate)}
          {item.outsidePlanYear && <OutsideYearNote year={item.plan.year} compact />}
        </span>
      ),
      budget: <CampaignBudgetCell campaign={item} />,
    },
    children: item.initiatives.map((initiative) => initiativeNode(initiative, options)),
    actions: [
      {
        label: ADD_INITIATIVE,
        onSelect: () => void options.navigate(addInitiativeHref(options.planId, item.id)),
      },
      ...viewAndEdit('campaign', item.id, options),
      ...deleteItem(item, options),
    ],
  }));
}

function initiativeNode(initiative: Initiative, options: RowOptions): TreeNode {
  return {
    id: initiative.id,
    level: 'initiative',
    name: initiative.name,
    href: recordHref('initiative', initiative.id, options.planId),
    highlighted: initiative.id === options.highlightId,
    cells: {
      ...commonCells(initiative, options),
      name: `${initiative.id} · ${activitiesCount(initiative.counts.activities)}`,
      dates: options.formatters.dateRange(initiative.startDate, initiative.endDate),
      budget: options.formatters.money(initiative.budget),
    },
    children: options.rowsBelow?.get(initiative.id) ?? [],
    actions: [
      ...viewAndEdit('initiative', initiative.id, options),
      ...initiativeExtraActions(
        { id: initiative.id, name: initiative.name, planId: options.planId },
        options.navigate,
      ),
      ...deleteItem(initiative, options),
    ],
  };
}

/** The cells both levels show the same way. */
function commonCells(record: Campaign | Initiative, options: RowOptions) {
  const level = 'initiatives' in record ? 'campaign' : 'initiative';
  const { budgetAndCost } = record;
  return {
    owner: record.owner ? <PersonLabel person={record.owner} /> : '—',
    status: (
      <span className={styles.chips}>
        <ApprovalProgressChip progress={record.approvalProgress} />
        <TimingChip timing={record.timing} />
        <CompletenessChip
          missing={record.completeness.missing.map((field) => ({
            id: field.field,
            label: field.label,
          }))}
          onSelectField={(field) =>
            void options.navigate(fieldHref(level, record.id, field, options.planId))
          }
          itemsBelow={record.completeness.itemsBelowNeedingAttention}
          onItemsBelowSelect={() =>
            void options.navigate(recordHref(level, record.id, options.planId))
          }
        />
      </span>
    ),
    plannedCost: options.formatters.money(budgetAndCost.plannedCost),
    budgetUsed: (
      <BudgetUsed percent={budgetAndCost.budgetUsedPercent} overBudget={budgetAndCost.overBudget} />
    ),
  };
}

function viewAndEdit(level: RecordLevel, id: string, options: RowOptions): ActionsMenuItem[] {
  const href = recordHref(level, id, options.planId);
  return [
    { label: VIEW, onSelect: () => void options.navigate(href) },
    { label: EDIT, onSelect: () => void options.navigate(`${href}/edit`) },
  ];
}

/** Delete, last and after a line (design system §6.1). */
function deleteItem(record: Campaign | Initiative, options: RowOptions): ActionsMenuItem[] {
  const { onDelete } = options;
  if (!onDelete) {
    return [];
  }
  return [
    {
      label: DELETE,
      tone: 'danger',
      separatorBefore: true,
      onSelect: () => onDelete(targetOf(record)),
    },
  ];
}

export function addInitiativeHref(planId: string, campaignId: string): string {
  return `${recordHref('campaign', campaignId, planId)}/initiatives/new`;
}

/**
 * The totals row (FR-022): "Total · 2 campaigns · 4 initiatives · 7 activities", the sum of the
 * campaign budgets ("Allocated to campaigns") and the plan's planned cost.
 */
export function totalsFooter(data: PlanCampaigns, formatters: Formatters): TreeTableFooter {
  const { totals } = data;
  return {
    name: totalsText(totals.campaigns, totals.initiatives, totals.activities),
    budget: (
      <span className={styles.money}>
        <span>{formatters.money(totals.allocatedToCampaigns)}</span>
        <span className={styles.left}>{ALLOCATED_TO_CAMPAIGNS}</span>
      </span>
    ),
    plannedCost: formatters.money(totals.plannedCost),
  };
}
