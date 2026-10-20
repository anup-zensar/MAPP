import { useNavigate } from 'react-router';
import type { components } from '@/api/schema';
import { Avatar, ProgressBar, StatusChip } from '@/components';
import { AttentionChip } from '@/features/campaigns/AttentionChip';
import { fieldHref, recordHref } from '@/lib/recordLinks';
import type { PlanningRecordType } from '@/lib/recordLinks';
import { useFormatters } from '@/lib/useFormatters';
import styles from './PlanFacts.module.css';
import { fieldsNeededText, groupLabel, STATUS } from './planText';
import type { PlanStatus } from './planText';

type Completeness = components['schemas']['Completeness'];
type OptionRef = components['schemas']['OptionRef'];
type PersonRef = components['schemas']['PersonRef'];
type PlanFigures = components['schemas']['PlanFigures'];

/** "Draft" or "Published" as a chip. */
export function PlanStatusChip({ status }: { status: PlanStatus }) {
  return <StatusChip tone={STATUS[status].tone}>{STATUS[status].word}</StatusChip>;
}

/**
 * The plan's completeness (FR-017) as one chip (design system §4.1): "All filled in", or
 * "<n> fields needed" / "<n> items below need attention" whose list opens the edit form on each
 * field and links to each record below.
 */
export function PlanCompleteness({
  planId,
  completeness,
  compact = false,
}: {
  planId: string;
  completeness: Completeness;
  /** The plan tree's row: a short chip ("2 items below"). */
  compact?: boolean;
}) {
  const navigate = useNavigate();
  return (
    <AttentionChip
      missing={completeness.missing.map((field) => ({ id: field.field, label: field.label }))}
      onSelectField={(field) => void navigate(fieldHref('plan', planId, field))}
      itemsBelow={completeness.itemsBelow.map(({ record, missing }) => ({
        id: `${record.type}:${record.id}`,
        label: `${groupLabel(record)} — ${fieldsNeededText(missing.length)}`,
        href: recordHref(record.type as PlanningRecordType, record.id, planId),
      }))}
      compact={compact}
    />
  );
}

/** "Owner: Priya Raman" with the avatar, "(inactive)" when they left; or "No owner yet". */
export function PlanOwner({ owner }: { owner: PersonRef | null }) {
  if (!owner) {
    return <span className={styles.muted}>No owner yet</span>;
  }
  return (
    <span className={styles.owner}>
      <Avatar initials={owner.initials} size="sm" />
      <span>
        Owner: {owner.name}
        {!owner.active && ' (inactive)'}
      </span>
    </span>
  );
}

/** "Business unit: Lubricants, Mobility" or "No business unit yet". */
export function BusinessUnitText({ units }: { units: OptionRef[] }) {
  if (units.length === 0) {
    return <span className={styles.muted}>No business unit yet</span>;
  }
  return (
    <span>
      Business unit:{' '}
      {units.map((unit) => (unit.active ? unit.label : `${unit.label} (inactive)`)).join(', ')}
    </span>
  );
}

/**
 * Planned cost against the plan budget (FR-002): the budget-used bar, red above 100 %, and
 * "Planned cost <a> · Budget <b> · <p> % used" or "Planned cost <a> · No budget set".
 */
export function BudgetUsed({ figures }: { figures: PlanFigures }) {
  const { money } = useFormatters();
  const planned = `Planned cost ${money(figures.plannedCost)}`;
  return (
    <div className={styles.budget}>
      {figures.budget ? (
        <>
          <ProgressBar
            value={figures.budgetUsedPercent}
            label="Budget used"
            tone={figures.overBudget ? 'problem' : undefined}
          />
          <p className={figures.overBudget ? styles.problem : undefined}>
            {planned} · Budget {money(figures.budget)}
            {figures.budgetUsedPercent !== null && ` · ${figures.budgetUsedPercent}% used`}
          </p>
        </>
      ) : (
        <p>{planned} · No budget set</p>
      )}
    </div>
  );
}
