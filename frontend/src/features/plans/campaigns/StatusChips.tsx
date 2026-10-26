import { TriangleAlert } from 'lucide-react';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import { Tooltip } from '@/ui/Tooltip/Tooltip';
import type { components } from '@/api/generated/schema';
import styles from './StatusChips.module.css';
import {
  activitiesApprovedOf,
  ALL_APPROVED,
  approvedOf,
  NO_ACTIVITIES,
  outsidePlanYear,
  TIMING_WORD,
} from './texts';

type ApprovalSummary = components['schemas']['ApprovalSummary'];
export type Timing = 'notStarted' | 'running' | 'finished' | null;

/**
 * Approval progress of the activities below (FR-013, design system §4.1): "No activities" (grey),
 * "3 of 5 approved" (amber, the waiting family; never "In progress", which means a running
 * activity — functional review A-02) and "All approved" (green). Screen readers hear the whole
 * meaning: "3 of 5 activities approved".
 */
export function ApprovalProgressChip({ progress }: { progress: ApprovalSummary }) {
  switch (progress.state) {
    case 'noActivities':
      return <StatusChip tone="neutral">{NO_ACTIVITIES}</StatusChip>;
    case 'allApproved':
      return <StatusChip tone="positive">{ALL_APPROVED}</StatusChip>;
    default:
      return (
        <StatusChip tone="review">
          <span aria-hidden="true">{approvedOf(progress.approved, progress.total)}</span>
          <span className="visually-hidden">
            {activitiesApprovedOf(progress.approved, progress.total)}
          </span>
        </StatusChip>
      );
  }
}

/**
 * Timing against today (FR-013) as words under a record's dates: Not started, Running, Finished.
 * Nothing while the record has no dates (its dates already read "—"). It is a fact about the
 * dates, not a status, so it is text, not a third chip (design system §4.1: at most two chips).
 */
export function TimingNote({ timing }: { timing: Timing }) {
  if (!timing) {
    return null;
  }
  return <span className={styles.timing}>{TIMING_WORD[timing]}</span>;
}

/**
 * R4 (FR-008): the campaign runs outside its plan's year. `compact` is the row's icon with the
 * text in a tooltip; otherwise the text is shown with the icon. A warning only: it never blocks.
 */
export function OutsideYearNote({ year, compact = false }: { year: number; compact?: boolean }) {
  const text = outsidePlanYear(year);
  if (compact) {
    return (
      <Tooltip content={text}>
        {/* eslint-disable-next-line jsx-a11y-x/no-noninteractive-tabindex -- opens the tooltip */}
        <span role="img" aria-label={text} tabIndex={0} className={styles.warningIcon}>
          <TriangleAlert aria-hidden="true" />
        </span>
      </Tooltip>
    );
  }
  return (
    <p className={styles.warning}>
      <TriangleAlert aria-hidden="true" />
      {text}
    </p>
  );
}
