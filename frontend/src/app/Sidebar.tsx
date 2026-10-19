import { ChevronLeft, ChevronRight } from 'lucide-react';
import { useId } from 'react';
import { Link, NavLink } from 'react-router';
import { Logo, Tooltip } from '@/components';
import { cx } from '@/lib/cx';
import { AccountMenu } from './AccountMenu';
import { MAIN_ITEMS, MORE_ITEMS } from './navigation';
import type { NavItem } from './navigation';
import { NotificationBell } from './NotificationBell';
import styles from './Sidebar.module.css';
import { ThemeToggle } from './ThemeToggle';
import { useCurrentUser, useSession } from './useSession';

export interface SidebarProps {
  /**
   * Inside the menu drawer of narrow screens: always expanded, without Collapse, and with
   * written-out names instead of tooltips, so one Escape always closes the drawer.
   */
  inDrawer?: boolean;
}

/**
 * The left sidebar (FR-010, FR-011, design system §2): the logo, the sections with a MORE group,
 * and at the bottom Collapse, the account button, the bell and the light/dark switch. Collapsed,
 * it is a 72 px icon rail; the person's choice is saved for them.
 */
export function Sidebar({ inDrawer = false }: SidebarProps) {
  const person = useCurrentUser();
  const { updatePreferences } = useSession();
  const moreId = useId();
  const collapsed = !inDrawer && person.preferences.sidebarCollapsed;
  const variant = inDrawer ? 'drawer' : collapsed ? 'rail' : 'full';

  function toggleCollapsed() {
    updatePreferences({ sidebarCollapsed: !collapsed });
  }

  return (
    <div className={cx(styles.sidebar, styles[variant])}>
      <div className={styles.brand}>
        <Link to="/" className={styles.homeLink} aria-label="NorthStar home">
          <Logo withWordmark={!collapsed} />
        </Link>
      </div>

      <nav aria-label="Main" className={styles.nav}>
        <NavItems items={MAIN_ITEMS} collapsed={collapsed} />
        {/* In the icon rail the MORE label becomes a hairline; screen readers still hear it. */}
        <p id={moreId} className={collapsed ? 'visually-hidden' : styles.groupLabel}>
          More
        </p>
        {collapsed && <div className={styles.divider} />}
        <NavItems items={MORE_ITEMS} collapsed={collapsed} labelledBy={moreId} />
      </nav>

      <div className={styles.footer}>
        {!inDrawer && <CollapseButton collapsed={collapsed} onToggle={toggleCollapsed} />}
        <div className={styles.controls}>
          <AccountMenu variant={variant} />
          <div className={styles.icons}>
            <NotificationBell withLabel={inDrawer} />
            <ThemeToggle withLabel={inDrawer} />
          </div>
        </div>
      </div>
    </div>
  );
}

interface NavItemsProps {
  items: NavItem[];
  collapsed: boolean;
  /** The id of the group's label, which names the list for screen readers. */
  labelledBy?: string;
}

/** One group of sidebar links. */
function NavItems({ items, collapsed, labelledBy }: NavItemsProps) {
  return (
    <ul className={styles.list} aria-labelledby={labelledBy}>
      {items.map((item) => (
        <li key={item.path}>
          <NavItemLink item={item} collapsed={collapsed} />
        </li>
      ))}
    </ul>
  );
}

/**
 * A sidebar link; NavLink marks the current section with aria-current="page", which also gives it
 * its highlight (Sidebar.module.css). Home is current only on Home itself ("end"); a section is
 * current on its sub-pages too.
 */
function NavItemLink({ item, collapsed }: { item: NavItem; collapsed: boolean }) {
  // A plain class name, not NavLink's className function: in the icon rail the link is the
  // trigger of a Tooltip, which joins class names as text and would turn a function into text.
  const link = (
    <NavLink to={item.path} end={item.path === '/'} className={styles.item}>
      <item.icon className={styles.icon} />
      {/* Hidden on screen in the icon rail, never removed: the link keeps its name. */}
      <span className={collapsed ? 'visually-hidden' : undefined}>{item.label}</span>
    </NavLink>
  );

  // In the icon rail a tooltip shows the hidden label on hover and focus.
  return collapsed ? (
    <Tooltip content={item.label} side="right">
      {link}
    </Tooltip>
  ) : (
    link
  );
}

/** "Collapse" (chevron and word), or in the icon rail the chevron alone with "Expand sidebar". */
function CollapseButton({ collapsed, onToggle }: { collapsed: boolean; onToggle: () => void }) {
  if (collapsed) {
    return (
      <Tooltip content="Expand sidebar" side="right">
        <button
          type="button"
          className={styles.collapse}
          aria-label="Expand sidebar"
          onClick={onToggle}
        >
          <ChevronRight />
        </button>
      </Tooltip>
    );
  }
  return (
    <button type="button" className={styles.collapse} onClick={onToggle}>
      <ChevronLeft />
      Collapse
    </button>
  );
}
