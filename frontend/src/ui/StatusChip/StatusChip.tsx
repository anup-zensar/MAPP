import type { ReactNode } from 'react';
import { cx } from '@/lib/cx';
import styles from './StatusChip.module.css';

/**
 * The status families of design system §4: neutral (Draft), review (Submitted), positive
 * (Approved), active (In progress), done (Complete) and problem (Declined).
 */
export type StatusTone = 'neutral' | 'review' | 'positive' | 'active' | 'done' | 'problem';

export interface StatusChipProps {
  tone: StatusTone;
  /** The status as a word ("Draft"): colour is never the only signal (constitution III). */
  children: ReactNode;
}

/** A record's status: its word on a light tint of its family's colour (design system §4). */
export function StatusChip({ tone, children }: StatusChipProps) {
  return <span className={cx(styles.chip, styles[tone])}>{children}</span>;
}
