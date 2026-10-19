import { Link } from 'react-router';
import { Logo } from '@/components';
import styles from './MobileTopBar.module.css';
import { NavDrawer } from './NavDrawer';
import { NotificationBell } from './NotificationBell';
import { ThemeToggle } from './ThemeToggle';

/**
 * The bar at the top of narrow screens (below 768 px), where there is no sidebar: Open menu, the
 * logo, the bell and the light/dark switch (plan "Layout"). It stays at the top while the page
 * scrolls.
 */
export function MobileTopBar() {
  return (
    <header className={styles.bar}>
      <NavDrawer />
      <Link to="/" className={styles.homeLink} aria-label="NorthStar home">
        <Logo withWordmark />
      </Link>
      <div className={styles.end}>
        <NotificationBell />
        <ThemeToggle />
      </div>
    </header>
  );
}
