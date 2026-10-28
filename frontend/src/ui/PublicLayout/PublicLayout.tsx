import type { ReactNode } from 'react';
import { DemoBanner } from '../DemoBanner/DemoBanner';
import { Logo } from '../Logo/Logo';
import styles from './PublicLayout.module.css';
import { ThemeToggle } from '../ThemeToggle/ThemeToggle';

export interface PublicLayoutProps {
  children: ReactNode;
  /** The error page leaves the switch out: the session it needs may be what failed. */
  withThemeSwitch?: boolean;
}

export function PublicLayout({ children, withThemeSwitch = true }: PublicLayoutProps) {
  return (
    <div className={styles.layout}>
      <a href="#main" className="skip-link">
        Skip to main content
      </a>
      <DemoBanner />
      <header className={styles.header}>
        <Logo />
        {withThemeSwitch && <ThemeToggle />}
      </header>
      {/* tabIndex -1 lets the skip link move the keyboard focus here. */}
      <main id="main" tabIndex={-1} className={styles.main}>
        {children}
      </main>
    </div>
  );
}
