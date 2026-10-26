import { CircleAlert, CircleCheck, Info, TriangleAlert } from 'lucide-react';
import type { ReactNode } from 'react';
import { cx } from '@/ui/cx';
import styles from './Notice.module.css';

export type NoticeTone = 'info' | 'warning' | 'success' | 'danger';

const ICONS = { info: Info, warning: TriangleAlert, success: CircleCheck, danger: CircleAlert };

export interface NoticeProps {
  tone: NoticeTone;
  /** A short bold first line ("Declined by Daniel Okoro on 15 Oct 2026, 14:41"). */
  title?: string;
  children?: ReactNode;
  /** "alert" for a failure that must be read out at once; "status" otherwise. */
  role?: 'status' | 'alert';
  /** Anchor for links such as #notice. */
  id?: string;
}

/**
 * A message inside the page (design system §5.6 `InlineAlert`): the feedback colour's tint, a 4 px
 * start bar and an icon, the optional title in bold and the text in ink. Used directly under the
 * page header for a declined or waiting record (UI audit AC-02), so the reason is never buried. It
 * lives here until the shared `InlineAlert` component exists.
 */
export function Notice({ tone, title, children, role = 'status', id }: NoticeProps) {
  const Icon = ICONS[tone];
  return (
    <div id={id} role={role} data-notice={tone} className={cx(styles.notice, styles[tone])}>
      <Icon aria-hidden="true" className={styles.icon} />
      <div className={styles.text}>
        {title && <p className={styles.title}>{title}</p>}
        {children}
      </div>
    </div>
  );
}
