import { useEffect, useRef, useState } from 'react';
import type { ReactNode } from 'react';

interface ScrollBoxProps {
  /** The table's name: the box is then named "<label>, scrolls sideways". */
  label: string;
  className?: string;
  children: ReactNode;
}

/**
 * The sideways scroll box of a table (design system §5.4): while the table is wider than the box,
 * the box is a named region in the tab order, so the keyboard can scroll it (WCAG 2.1.1); the
 * edge shadows are the CSS of `className`. Used by the Financials and global budget tables.
 */
export function ScrollBox({ label, className, children }: ScrollBoxProps) {
  const ref = useRef<HTMLDivElement>(null);
  const [overflows, setOverflows] = useState(false);

  useEffect(() => {
    const box = ref.current;
    if (!box || typeof ResizeObserver === 'undefined') {
      return;
    }
    const observer = new ResizeObserver(() => {
      setOverflows(box.scrollWidth > box.clientWidth + 1);
    });
    observer.observe(box);
    if (box.firstElementChild) {
      observer.observe(box.firstElementChild);
    }
    return () => observer.disconnect();
  }, []);

  return (
    <div
      ref={ref}
      className={className}
      {...(overflows
        ? { tabIndex: 0, role: 'region', 'aria-label': `${label}, scrolls sideways` }
        : {})}
    >
      {children}
    </div>
  );
}
