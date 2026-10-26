import type { ReactNode } from 'react';
import { cx } from '@/ui/cx';
import styles from './ScrollBox.module.css';
import { useSideScroll } from './useSideScroll';

export interface ScrollBoxProps {
  /** The table's name: the box is "<label>, scrolls sideways" while it overflows. */
  label: string;
  children: ReactNode;
}

/**
 * The box a wide All work table scrolls in, never the page (design system §5.4, UI audit AW-05):
 * a shadow on each edge that hides columns, and — only while it overflows — a named region in the
 * tab order, so keyboard users can scroll it with the arrow keys.
 */
export function ScrollBox({ label, children }: ScrollBoxProps) {
  const { ref, scroll } = useSideScroll<HTMLDivElement>();
  const overflows = scroll.start || scroll.end;

  return (
    <div className={cx(styles.frame, scroll.start && styles.cueStart, scroll.end && styles.cueEnd)}>
      <div
        ref={ref}
        className={styles.scroll}
        {...(overflows
          ? { role: 'region', 'aria-label': `${label}, scrolls sideways`, tabIndex: 0 }
          : {})}
      >
        {children}
      </div>
    </div>
  );
}
