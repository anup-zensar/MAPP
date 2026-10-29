import { ChevronDown, ChevronRight } from 'lucide-react';
import { useState } from 'react';
import type { ReactNode } from 'react';
import { FieldHelpTip } from '@/features/settings';
import { formatPercent } from '@/format/decimal';
import { useFormatters } from '@/format/useFormatters';
import { cx } from '@/ui/cx';
import gridStyles from './AllWorkGrid.module.css';
import type { AllWorkBreakdown, BreakdownGroup, BreakdownSubgroup } from './all-work-types';
import type { GroupRef } from './breakdown-state';
import styles from './BreakdownTable.module.css';
import { COLUMN_LABELS, groupToggleText, showInGridText } from './breakdown-text';
import { BREAKDOWN_SHARE_HELP_KEY } from './column-help';
import { ScrollBox } from './ScrollBox';

type OpenGroup = (groups: GroupRef[]) => (() => void) | null;

export interface BreakdownTableProps {
  breakdown: AllWorkBreakdown;
  caption: string;
  /** What opening a group does; null when it cannot be opened. Without it names are plain text. */
  openGroup?: OpenGroup;
}

/** The groups of a Breakdown with their then-by groups behind a toggle, the totals row and notes. */
export function BreakdownTable({ breakdown, caption, openGroup }: BreakdownTableProps) {
  const [open, setOpen] = useState<string[]>([]);
  const budget = breakdown.by.hasBudget;
  const heading = breakdown.thenBy
    ? `${breakdown.by.label} › ${breakdown.thenBy.label}`
    : breakdown.by.label;

  return (
    <div className={gridStyles.grid}>
      <ScrollBox label={caption}>
        <table className={gridStyles.table}>
          <caption className="visually-hidden">{caption}</caption>
          <thead>
            <tr>
              <th scope="col" className={gridStyles.nameHead}>
                {heading}
              </th>
              {columnsOf(budget).map((label) => (
                <th key={label} scope="col" className={gridStyles.end}>
                  {label}
                  {label === COLUMN_LABELS.share && (
                    <FieldHelpTip helpKey={BREAKDOWN_SHARE_HELP_KEY} label={label} />
                  )}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {breakdown.groups.map((group) => (
              <GroupRows
                key={group.key}
                group={group}
                budget={budget}
                expanded={open.includes(group.key)}
                onToggle={() =>
                  setOpen((keys) =>
                    keys.includes(group.key)
                      ? keys.filter((key) => key !== group.key)
                      : [...keys, group.key],
                  )
                }
                openGroup={openGroup}
              />
            ))}
          </tbody>
          <tfoot>
            <Row group={breakdown.total} budget={budget} className={gridStyles.totals}>
              {breakdown.total.label}
            </Row>
          </tfoot>
        </table>
      </ScrollBox>
      {breakdown.notes.map((note) => (
        <p key={note} className={styles.note}>
          {note}
        </p>
      ))}
    </div>
  );
}

function columnsOf(budget: boolean): string[] {
  const columns = [
    COLUMN_LABELS.activities,
    COLUMN_LABELS.lineItems,
    COLUMN_LABELS.plannedCost,
    COLUMN_LABELS.share,
  ];
  return budget ? [...columns, COLUMN_LABELS.budget, COLUMN_LABELS.budgetUsed] : columns;
}

interface GroupRowsProps {
  group: BreakdownGroup;
  budget: boolean;
  expanded: boolean;
  onToggle: () => void;
  openGroup: OpenGroup | undefined;
}

function GroupRows({ group, budget, expanded, onToggle, openGroup }: GroupRowsProps) {
  const hasChildren = group.children.length > 0;
  return (
    <>
      <Row group={group} budget={budget}>
        <span className={styles.name}>
          {hasChildren && (
            <button
              type="button"
              className={styles.toggle}
              aria-expanded={expanded}
              aria-label={groupToggleText(group.label, expanded)}
              onClick={onToggle}
            >
              {expanded ? <ChevronDown aria-hidden="true" /> : <ChevronRight aria-hidden="true" />}
            </button>
          )}
          <GroupName groups={[group]} openGroup={openGroup} />
        </span>
      </Row>
      {expanded &&
        group.children.map((child) => (
          <Row key={child.key} group={child} budget={budget} className={styles.child}>
            <GroupName groups={[group, child]} openGroup={openGroup} />
          </Row>
        ))}
    </>
  );
}

function GroupName({ groups, openGroup }: { groups: GroupRef[]; openGroup?: OpenGroup }) {
  const label = groups.at(-1)?.label ?? '';
  const onOpen = openGroup?.(groups);
  if (!onOpen) {
    return label;
  }
  return (
    <button type="button" className={styles.open} title={showInGridText(label)} onClick={onOpen}>
      {label}
    </button>
  );
}

interface RowProps {
  group: BreakdownGroup | BreakdownSubgroup;
  budget: boolean;
  className?: string;
  children: ReactNode;
}

function Row({ group, budget, className, children }: RowProps) {
  const { money } = useFormatters();
  return (
    <tr className={className}>
      <th scope="row" className={gridStyles.nameCell}>
        {children}
      </th>
      <td className={gridStyles.end}>{group.activities}</td>
      <td className={gridStyles.end}>{group.lineItems}</td>
      <td className={gridStyles.end}>{money(group.plannedCost)}</td>
      <td className={gridStyles.end}>{formatPercent(group.share)}</td>
      {budget && <td className={gridStyles.end}>{money(group.budget)}</td>}
      {budget && (
        <td className={cx(gridStyles.end, group.overBudget && styles.over)}>
          {group.budgetUsedPercent === null ? '—' : `${group.budgetUsedPercent} %`}
        </td>
      )}
    </tr>
  );
}
