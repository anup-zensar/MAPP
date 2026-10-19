import { useId, useRef } from 'react';
import type { KeyboardEvent, ReactNode } from 'react';
import { cx } from '@/lib/cx';
import styles from './PillTabs.module.css';

/** One tab. */
export interface PillTab<T extends string> {
  value: T;
  label: string;
  /** Shown after the label, usually a CountBadge. */
  badge?: ReactNode;
}

export interface PillTabsProps<T extends string> {
  /** Names the tab list for screen readers: "Approvals". */
  label: string;
  tabs: PillTab<T>[];
  value: T;
  onChange: (value: T) => void;
  /** What the chosen tab shows: the tab panel's content. */
  children: ReactNode;
}

/**
 * Tabs as pills (feature 004; Approvals' Waiting for me · All open · Decided by me, All work's
 * views): an ARIA tab list with one tab in the tab order; ← → Home End move to another tab and
 * show it. The content below is the chosen tab's panel.
 */
export function PillTabs<T extends string>({
  label,
  tabs,
  value,
  onChange,
  children,
}: PillTabsProps<T>) {
  const id = useId();
  const list = useRef<HTMLDivElement>(null);
  const tabId = (tab: T) => `${id}-tab-${tab}`;
  const panelId = `${id}-panel`;

  function choose(index: number) {
    const tab = tabs[(index + tabs.length) % tabs.length];
    onChange(tab.value);
    list.current?.querySelector<HTMLElement>(`[data-tab="${tab.value}"]`)?.focus();
  }

  function onKeyDown(event: KeyboardEvent<HTMLButtonElement>, index: number) {
    const moves: Record<string, number> = {
      ArrowRight: index + 1,
      ArrowLeft: index - 1,
      Home: 0,
      End: tabs.length - 1,
    };
    if (event.key in moves) {
      event.preventDefault();
      choose(moves[event.key]);
    }
  }

  return (
    <div className={styles.tabs}>
      <div ref={list} role="tablist" aria-label={label} className={styles.list}>
        {tabs.map((tab, index) => {
          const selected = tab.value === value;
          return (
            <button
              key={tab.value}
              id={tabId(tab.value)}
              data-tab={tab.value}
              type="button"
              role="tab"
              aria-selected={selected}
              aria-controls={panelId}
              tabIndex={selected ? 0 : -1}
              className={cx(styles.tab, selected && styles.selected)}
              onClick={() => onChange(tab.value)}
              onKeyDown={(event) => onKeyDown(event, index)}
            >
              {tab.label}
              {tab.badge && <> {tab.badge}</>}
            </button>
          );
        })}
      </div>
      <div id={panelId} role="tabpanel" aria-labelledby={tabId(value)} className={styles.panel}>
        {children}
      </div>
    </div>
  );
}
