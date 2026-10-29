import { ShieldAlert } from 'lucide-react';
import type { OverspendMode } from './api';
import { REFUSES_OVERSPENDING } from './overspend-text';
import styles from './RefusesOverspending.module.css';

/** "Refuses overspending" next to a budget set to Refuse (037 FR-002); nothing for Warn. */
export function RefusesOverspending({ mode }: { mode: OverspendMode }) {
  if (mode !== 'refuse') {
    return null;
  }
  return (
    <span className={styles.tag}>
      <ShieldAlert aria-hidden="true" />
      {REFUSES_OVERSPENDING}
    </span>
  );
}
