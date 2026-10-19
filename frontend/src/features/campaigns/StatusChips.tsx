import { TriangleAlert } from 'lucide-react';
import { InfoTip, StatusChip, Tooltip } from '@/components';
import type { components } from '@/api/schema';
import styles from './StatusChips.module.css';
import {
  ALL_APPROVED,
  approvedOf,
  IN_PROGRESS,
  NO_ACTIVITIES,
  outsidePlanYear,
  TIMING_TIP,
  TIMING_WORD,
} from './texts';

type ApprovalSummary = components['schemas']['ApprovalSummary'];
export type Timing = 'notStarted' | 'running' | 'finished' | null;

/**
 * Approval progress of the activities below (FR-013, design system §4): "No activities" (grey),
 * "In progress · 3 of 5 activities approved" (amber), "All approved" (green).
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
          {IN_PROGRESS} · {approvedOf(progress.approved, progress.total)}
        </StatusChip>
      );
  }
}

const TIMING_TONE = { notStarted: 'neutral', running: 'active', finished: 'done' } as const;

/**
 * Timing against today (FR-013): Not started, Running, Finished; "—" with the reason while the
 * record has no dates.
 */
export function TimingChip({ timing }: { timing: Timing }) {
  if (!timing) {
    return (
      <span className={styles.noTiming}>
        <span>—</span>
        <InfoTip label="Timing" text={TIMING_TIP} />
      </span>
    );
  }
  return <StatusChip tone={TIMING_TONE[timing]}>{TIMING_WORD[timing]}</StatusChip>;
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
