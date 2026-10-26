import type { ReactNode } from 'react';
import { DemoBanner } from '../DemoBanner/DemoBanner';
import { Logo } from '../Logo/Logo';
import styles from './PublicLayout.module.css';
import { ThemeToggle } from '../ThemeToggle/ThemeToggle';

export interface PublicLayoutProps {
  children: ReactNode;
  /**
   * Shows the light/dark switch on the right of the header (the default). The error page leaves
   * it out: it is shown when the app itself failed, possibly the session the switch needs.
   */
  withThemeSwitch?: boolean;
}

/**
 * The frame of the pages people see without signing in: the sign-in page, "Page not found" and
 * the full-page loading and error states (plan "Layout"). From top to bottom: the skip link (the
 * first thing the keyboard reaches), the demo banner across the whole width, a header with the
 * logo and the theme switch, and the page itself in <main id="main">.
 */
export function PublicLayout({ children, withThemeSwitch = true }: PublicLayoutProps) {
  return (
    <div className={styles.layout}>
      <a href="#main" className="skip-link">
        Skip to main content
      </a>
      <DemoBanner />
      <header className={styles.header}>
        <Logo withWordmark />
        {withThemeSwitch && <ThemeToggle />}
      </header>
      {/* tabIndex -1 lets the skip link move the keyboard focus here. */}
      <main id="main" tabIndex={-1} className={styles.main}>
        {children}
      </main>
    </div>
  );
}
