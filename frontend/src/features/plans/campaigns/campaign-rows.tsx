import { Plus } from 'lucide-react';
import type { NavigateFunction } from 'react-router';
import { PersonLabel } from '@/ui/PersonLabel/PersonLabel';
import { Tooltip } from '@/ui/Tooltip/Tooltip';
import type { ActionsMenuItem } from '@/ui/ActionsMenu/ActionsMenu';
import type { TreeNode } from '@/ui/TreeTable/TreeTable';
import { gateItem } from '@/session/gate-action';
import { recordHref } from '@/ui/record-links';
import type { Formatters } from '@/format/useFormatters';
import { completenessFieldHref } from '../phasing/phasing-links';
import { AttentionChip } from './AttentionChip';
import { BudgetCell, CampaignBudgetCell, PlannedCostCell } from './BudgetCells';
import styles from './campaign-rows.module.css';
import type { Campaign, Initiative, PlanCampaigns } from './api';
import { initiativeExtraActions } from './extensions';
import { targetOf } from './record-target';
import type { RecordTarget } from './record-target';
import { ApprovalProgressChip, OutsideYearNote, TimingNote } from './StatusChips';
import {
  activitiesCount,
  ADD_INITIATIVE,
  addInitiativeTo,
  COPY,
  DELETE,
  EDIT,
  initiativesCount,
  MOVE_TO,
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
          </span>
          {/* The R4 warning beside the timing, so a range over two years stays one line wide. */}
          <span className={styles.timingLine}>
            <TimingNote timing={item.timing} />
            {item.outsidePlanYear && <OutsideYearNote year={item.plan.year} compact />}
          </span>
        </span>
      ),
      budget: <CampaignBudgetCell campaign={item} />,
    },
    children: item.initiatives.map((initiative) => initiativeNode(initiative, options)),
    actions: [
      ...gateItem(
        {
          label: ADD_INITIATIVE,
          onSelect: () => void options.navigate(addInitiativeHref(options.planId, item.id)),
        },
        item.access?.addChild,
      ),
      ...viewAndEdit('campaign', item, options),
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
      ...viewAndEdit('initiative', initiative, options),
      ...copyAndMove(initiative, options),
      ...initiativeExtraActions(
        {
          id: initiative.id,
          name: initiative.name,
          planId: options.planId,
          addActivity: initiative.access?.addChild,
        },
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
            void options.navigate(completenessFieldHref(level, record.id, field, options.planId))
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

function viewAndEdit(
  level: RecordLevel,
  record: Campaign | Initiative,
  options: RowOptions,
): ActionsMenuItem[] {
  const href = recordHref(level, record.id, options.planId);
  return [
    { label: VIEW, onSelect: () => void options.navigate(href) },
    ...gateItem(
      { label: EDIT, onSelect: () => void options.navigate(`${href}/edit`) },
      record.access?.edit,
    ),
  ];
}

/** Copy and Move to… (US6), each listed only when the caller handles it. */
function copyAndMove(record: Campaign | Initiative, options: RowOptions): ActionsMenuItem[] {
  const { onCopy, onMove } = options;
  return [
    ...(onCopy
      ? gateItem({ label: COPY, onSelect: () => onCopy(targetOf(record)) }, record.access?.copy)
      : []),
    ...(onMove
      ? gateItem({ label: MOVE_TO, onSelect: () => onMove(targetOf(record)) }, record.access?.move)
      : []),
  ];
}

/** Delete, last and after a line (design system §6.1). */
function deleteItem(record: Campaign | Initiative, options: RowOptions): ActionsMenuItem[] {
  const { onDelete } = options;
  if (!onDelete) {
    return [];
  }
  return gateItem(
    {
      label: DELETE,
      tone: 'danger',
      separatorBefore: true,
      onSelect: () => onDelete(targetOf(record)),
    },
    record.access?.delete,
  );
}

export function addInitiativeHref(planId: string, campaignId: string): string {
  return `${recordHref('campaign', campaignId, planId)}/initiatives/new`;
}
