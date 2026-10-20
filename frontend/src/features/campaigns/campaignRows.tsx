import { Plus } from 'lucide-react';
import type { NavigateFunction } from 'react-router';
import { PersonLabel, Tooltip } from '@/components';
import type { ActionsMenuItem, TreeNode, TreeTableFooter } from '@/components';
import { AmountPair } from '@/features/plans/AmountPair';
import { fieldHref, recordHref } from '@/lib/recordLinks';
import type { Formatters } from '@/lib/useFormatters';
import { AttentionChip } from './AttentionChip';
import { BudgetCell, CampaignBudgetCell, PlannedCostCell } from './BudgetCost';
import styles from './campaignRows.module.css';
import type { Campaign, Initiative, PlanCampaigns } from './campaignApi';
import { initiativeExtraActions } from './extensions';
import { ApprovalProgressChip, OutsideYearNote, TimingNote } from './StatusChips';
import {
  activitiesCount,
  ADD_INITIATIVE,
  addInitiativeTo,
  ALLOCATED_TO_CAMPAIGNS,
  COPY,
  DELETE,
  EDIT,
  initiativesCount,
  MOVE_TO,
  totalsText,
  VIEW,
} from './texts';
import type { RecordLevel } from './texts';

/**
 * The plan tree's cells, by column key (feature 004's PlanTree columns Name, Owner, Dates, Status,
 * Budget, Planned cost): `name` (shown after the name: ID, child count and the quick action),
 * `owner`, `dates`, `status`, `budget` (with what remains beneath) and `plannedCost` (with the
 * share of the budget it uses beneath).
 */
export const CELL_KEYS = ['name', 'owner', 'dates', 'status', 'budget', 'plannedCost'] as const;

/** A campaign or initiative a row action is about. */
export interface RecordTarget {
  level: RecordLevel;
  id: string;
  name: string;
  planId: string;
  /** The plan's year: a campaign moves only between plans of that year (FR-023). */
  year: number;
  /** The version the row or panel showed; a move sends it (FR-025). */
  version: number;
  /** What goes with it (the delete, move and copy dialogs). */
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
  /** **Copy** (US6): opens the copy dialog; the item is listed only with it. */
  onCopy?: (target: RecordTarget) => void;
  /** **Move to…** (US6): opens the move dialog; the item is listed only with it. */
  onMove?: (target: RecordTarget) => void;
}

/** The record a row stands for, for its actions. */
export function targetOf(record: Campaign | Initiative): RecordTarget {
  const common = {
    id: record.id,
    name: record.name,
    planId: record.planId,
    year: record.plan.year,
    version: record.version,
  };
  return 'initiatives' in record
    ? { ...common, level: 'campaign', counts: record.counts }
    : { ...common, level: 'initiative', counts: record.counts };
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
          {/* A 24 px target (WCAG 2.5.8) on the ID line, so the row keeps its 2-line height. */}
          <Tooltip content={addInitiativeTo(item.name)}>
            <button
              type="button"
              aria-label={addInitiativeTo(item.name)}
              className={styles.quickAction}
              onClick={() => void options.navigate(addInitiativeHref(options.planId, item.id))}
            >
              <Plus aria-hidden="true" />
            </button>
          </Tooltip>
        </span>
      ),
      dates: (
        <span className={styles.dates}>
          <span className={styles.nowrap}>
            {options.formatters.dateRange(item.startDate, item.endDate)}
            {item.outsidePlanYear && <OutsideYearNote year={item.plan.year} compact />}
          </span>
          <TimingNote timing={item.timing} />
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
      ...copyAndMove(item, options),
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
      dates: (
        <span className={styles.dates}>
          <span className={styles.nowrap}>
            {options.formatters.dateRange(initiative.startDate, initiative.endDate)}
          </span>
          <TimingNote timing={initiative.timing} />
        </span>
      ),
      budget: (
        <BudgetCell
          budget={initiative.budget}
          displayBudget={initiative.display.budget}
          remaining={null}
        />
      ),
    },
    children: options.rowsBelow?.get(initiative.id) ?? [],
    actions: [
      ...viewAndEdit('initiative', initiative.id, options),
      ...copyAndMove(initiative, options),
      ...initiativeExtraActions(
        { id: initiative.id, name: initiative.name, planId: options.planId },
        options.navigate,
      ),
      ...deleteItem(initiative, options),
    ],
  };
}

/**
 * The cells both levels show the same way. The Status column holds at most two chips (design
 * system §4.1): the approval roll-up and one completeness chip whose list holds the rest.
 */
function commonCells(record: Campaign | Initiative, options: RowOptions) {
  const level = 'initiatives' in record ? 'campaign' : 'initiative';
  // Amounts in the plan currency, the reader's display currency beneath when it differs (016
  // FR-013; before 016 both are the same).
  const { budgetAndCost } = record;
  return {
    owner: record.owner ? (
      <span className={styles.nowrap}>
        <PersonLabel person={record.owner} />
      </span>
    ) : (
      '—'
    ),
    status: (
      <span className={styles.chips}>
        <ApprovalProgressChip progress={record.approvalProgress} />
        <AttentionChip
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
          compact
        />
      </span>
    ),
    plannedCost: (
      <PlannedCostCell
        cost={budgetAndCost.plannedCost}
        displayCost={record.display.budgetAndCost.plannedCost}
        percent={budgetAndCost.budgetUsedPercent}
        overBudget={budgetAndCost.overBudget}
      />
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

/** Copy and Move to… (US6), each listed only when the caller handles it. */
function copyAndMove(record: Campaign | Initiative, options: RowOptions): ActionsMenuItem[] {
  const { onCopy, onMove } = options;
  return [
    ...(onCopy ? [{ label: COPY, onSelect: () => onCopy(targetOf(record)) }] : []),
    ...(onMove ? [{ label: MOVE_TO, onSelect: () => onMove(targetOf(record)) }] : []),
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
  // Amounts in the plan currency, the reader's display currency beneath when it differs (016
  // FR-013).
  const { totals, displayTotals } = data;
  return {
    // "Total" on the first line, the counts in one line beneath (UI audit PL-05: never wrapping).
    name: (
      <span className={styles.total}>
        <span>Total</span>
        <span className={styles.left}>
          {totalsText(totals.campaigns, totals.initiatives, totals.activities)}
        </span>
      </span>
    ),
    budget: (
      <span className={styles.money}>
        <AmountPair
          primary={totals.allocatedToCampaigns}
          secondary={displayTotals.allocatedToCampaigns}
        />
        <span className={styles.left}>{ALLOCATED_TO_CAMPAIGNS}</span>
      </span>
    ),
    plannedCost: totals.plannedCost ? (
      <AmountPair primary={totals.plannedCost} secondary={displayTotals.plannedCost} />
    ) : (
      formatters.money(null)
    ),
  };
}
