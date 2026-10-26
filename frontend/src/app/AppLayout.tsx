import type { ReactNode } from 'react';
import { DemoBanner } from '@/ui/DemoBanner/DemoBanner';
import { cx } from '@/ui/cx';
import styles from './AppLayout.module.css';
import { Sidebar } from './Sidebar';
import { TopBar } from './TopBar';
import { useCurrentUser } from '@/session/useSession';

/**
 * The frame of every signed-in page (FR-010, design system §2): the skip link first, the white
 * top bar across the whole width, the white sidebar on the left (from 960 px; narrower screens
 * open it as a drawer from the top bar), then the content column on the grey page with the demo
 * banner, the page in <main id="main"> (at most 1300 px wide, centred) and the footer.
 */
export function AppLayout({ children }: { children: ReactNode }) {
  const person = useCurrentUser();

  return (
    <div className={cx(styles.layout, person.preferences.sidebarCollapsed && styles.collapsed)}>
      <a href="#main" className="skip-link">
        Skip to main content
      </a>
      <TopBar />
      <div className={styles.sidebar}>
        <Sidebar />
      </div>
      <div className={styles.column}>
        <DemoBanner />
        {/* tabIndex -1 lets the skip link move the keyboard focus here. */}
        <main id="main" tabIndex={-1} className={styles.main}>
          <div className={styles.content}>{children}</div>
        </main>
        <footer className={styles.footer}>
          <p className={styles.footerText}>© MAPP — demo data</p>
        </footer>
      </div>
    </div>
  );
}
