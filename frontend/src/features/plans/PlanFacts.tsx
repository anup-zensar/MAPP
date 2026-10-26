import { useNavigate } from 'react-router';
import type { components } from '@/api/generated/schema';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import { AttentionChip } from './campaigns/AttentionChip';
import { completenessFieldHref } from './phasing/phasing-links';
import { recordHref } from '@/ui/record-links';
import type { PlanningRecordType } from '@/ui/record-links';
import styles from './PlanFacts.module.css';
import { fieldsNeededText, groupLabel, STATUS } from './plan-text';
import type { PlanStatus } from './plan-text';

type Completeness = components['schemas']['Completeness'];
type OptionRef = components['schemas']['OptionRef'];

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
      onSelectField={(field) => void navigate(completenessFieldHref('plan', planId, field, planId))}
      itemsBelow={completeness.itemsBelow.map(({ record, missing }) => ({
        id: `${record.type}:${record.id}`,
        label: `${groupLabel(record)} — ${fieldsNeededText(missing.length)}`,
        href: recordHref(record.type as PlanningRecordType, record.id, planId),
      }))}
      compact={compact}
    />
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
