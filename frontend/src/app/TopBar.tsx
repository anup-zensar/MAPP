import { Link } from 'react-router';
import { Logo } from '@/ui/Logo/Logo';
import { LogoTile } from '@/ui/Logo/LogoTile';
import { ThemeToggle } from '@/ui/ThemeToggle/ThemeToggle';
import { AccountMenu } from './AccountMenu';
import { NavDrawer } from './NavDrawer';
import { NotificationBell } from './NotificationBell';
import styles from './TopBar.module.css';

/** Below 960 px the name gives way to its tile, and Open menu comes before it. */
export function TopBar() {
  return (
    <header className={styles.bar}>
      <div className={styles.menu}>
        <NavDrawer />
      </div>
      <Link to="/" className={styles.homeLink} aria-label="SP&A One home">
        <span className={styles.name}>
          <Logo />
        </span>
        <span className={styles.tile}>
          <LogoTile />
        </span>
      </Link>
      <div className={styles.end}>
        <ThemeToggle />
        <NotificationBell />
        <AccountMenu />
      </div>
    </header>
  );
}
