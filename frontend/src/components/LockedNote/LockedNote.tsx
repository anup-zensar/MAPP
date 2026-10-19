import { Lock } from 'lucide-react';
import type { ReactNode } from 'react';
import styles from './LockedNote.module.css';

export interface LockedNoteProps {
  /** Why it can't change, and what to do instead (design system §6.6). */
  children: ReactNode;
  /** The id that a locked control points to with aria-describedby. */
  id?: string;
}

/**
 * The reason something can't change, next to it: a lock and one sentence ("The code can't change
 * once the currency is saved."). Nothing is ever greyed out without one (constitution III).
 */
export function LockedNote({ children, id }: LockedNoteProps) {
  return (
    <p id={id} className={styles.note}>
      <Lock aria-hidden="true" />
      <span>{children}</span>
    </p>
  );
}
