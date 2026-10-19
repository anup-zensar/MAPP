import type { ReactNode } from 'react';
import { DemoBanner } from '@/components';
import { cx } from '@/lib/cx';
import styles from './AppLayout.module.css';
import { MobileTopBar } from './MobileTopBar';
import { Sidebar } from './Sidebar';
import { useCurrentUser } from './useSession';

/**
 * The frame of every signed-in page (FR-010, design system §2): the skip link first, the sidebar
 * on the left (the top bar with Open menu below 768 px), then the content column with the demo
 * banner, the page in <main id="main"> (at most 1120 px wide) and the footer.
 */
export function AppLayout({ children }: { children: ReactNode }) {
  const person = useCurrentUser();

  return (
    <div className={cx(styles.layout, person.preferences.sidebarCollapsed && styles.collapsed)}>
      <a href="#main" className="skip-link">
        Skip to main content
      </a>
      <div className={styles.sidebar}>
        <Sidebar />
      </div>
      <div className={styles.column}>
        <DemoBanner />
        <MobileTopBar />
        {/* tabIndex -1 lets the skip link move the keyboard focus here. */}
        <main id="main" tabIndex={-1} className={styles.main}>
          <div className={styles.content}>{children}</div>
        </main>
        <footer className={styles.footer}>
          <p className={styles.footerText}>© NorthStar — demo data</p>
        </footer>
      </div>
    </div>
  );
}
