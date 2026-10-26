import { Link } from 'react-router';
import { Logo } from '@/ui/Logo/Logo';
import { AccountMenu } from './AccountMenu';
import { NavDrawer } from './NavDrawer';
import { NotificationBell } from './NotificationBell';
import { ThemeToggle } from '@/ui/ThemeToggle/ThemeToggle';
import styles from './TopBar.module.css';

/**
 * The white 56 px bar across the top of every signed-in page (FR-010, Shell's header): the
 * MAPP mark on the left — with Open menu before it on screens too narrow for the sidebar —
 * and on the right the light/dark switch, the bell and the account menu. It stays at the top
 * while the page scrolls.
 */
export function TopBar() {
  return (
    <header className={styles.bar}>
      <div className={styles.menu}>
        <NavDrawer />
      </div>
      <Link to="/" className={styles.homeLink} aria-label="MAPP home">
        <Logo withWordmark />
      </Link>
      <div className={styles.end}>
        <ThemeToggle />
        <NotificationBell />
        <AccountMenu />
      </div>
    </header>
  );
}
