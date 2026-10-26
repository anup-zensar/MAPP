import { CircleAlert, CircleCheck, Clock, Info } from 'lucide-react';
import type { ReactNode } from 'react';
import { cx } from '@/ui/cx';
import styles from './ApprovalAlert.module.css';

/** The feedback families of design system §5.6: what the message is about. */
export type ApprovalAlertTone = 'info' | 'warning' | 'success' | 'danger';

const ICONS = { info: Info, warning: Clock, success: CircleCheck, danger: CircleAlert };

export interface ApprovalAlertProps {
  tone: ApprovalAlertTone;
  /** A short bold first line: "Declined by Daniel Okoro on 12 Mar 2026, 14:32". */
  title?: string;
  children?: ReactNode;
}

/**
 * A message inside the page, drawn as design system §5.6's InlineAlert (a tinted box with a 4 px
 * bar and an icon in the feedback colour): the decline reason, the waiting notice, the delegation
 * banners. The words carry the meaning; the colour only repeats it. Announced politely, never as
 * an interruption, because it describes the page rather than a failure.
 */
export function ApprovalAlert({ tone, title, children }: ApprovalAlertProps) {
  const Icon = ICONS[tone];
  return (
    <div role="status" className={cx(styles.alert, styles[tone])}>
      <Icon aria-hidden="true" className={styles.icon} />
      <div className={styles.text}>
        {title && <p className={styles.title}>{title}</p>}
        {children}
      </div>
    </div>
  );
}
