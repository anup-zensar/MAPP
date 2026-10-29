import { Suspense } from 'react';
import type { ReactNode } from 'react';
import { DemoBanner } from '@/ui/DemoBanner/DemoBanner';
import { cx } from '@/ui/cx';
import { useCurrentUser } from '@/session/useSession';
import styles from './AppLayout.module.css';
import { PageLoading } from './PageLoading';
import { Sidebar } from './Sidebar';
import { TopBar } from './TopBar';

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
          <div className={styles.content}>
            <Suspense fallback={<PageLoading />}>{children}</Suspense>
          </div>
        </main>
        <footer className={styles.footer}>
          <p className={styles.footerText}>© SP&A One — demo data</p>
        </footer>
      </div>
    </div>
  );
}
