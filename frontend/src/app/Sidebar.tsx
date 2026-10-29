import { ChevronsLeft, ChevronsRight } from 'lucide-react';
import { Suspense, useId } from 'react';
import { Link, useLocation, useMatch } from 'react-router';
import { Logo } from '@/ui/Logo/Logo';
import { Tooltip } from '@/ui/Tooltip/Tooltip';
import { itemsFor, useAccess } from '@/session/access';
import { cx } from '@/ui/cx';
import { useCurrentUser, useSession } from '@/session/useSession';
import { MAIN_ITEMS, MORE_ITEMS } from './navigation';
import type { NavItem } from './navigation';
import styles from './Sidebar.module.css';

export interface SidebarProps {
  /** In the menu drawer: always expanded and without Collapse, so one Escape closes the drawer. */
  inDrawer?: boolean;
}

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
      {inDrawer && (
        <div className={styles.brand}>
          <Link to="/" className={styles.homeLink} aria-label="SP&A One home">
            <Logo />
          </Link>
        </div>
      )}

      <nav aria-label="Main" className={styles.nav}>
        <NavItems items={MAIN_ITEMS} collapsed={collapsed} />
        {/* In the icon rail the More label becomes a hairline; screen readers still hear it. */}
        <p id={moreId} className={collapsed ? 'visually-hidden' : styles.groupLabel}>
          More
        </p>
        {collapsed && <div className={styles.divider} />}
        <NavItems items={MORE_ITEMS} collapsed={collapsed} labelledBy={moreId} />
      </nav>

      {!inDrawer && (
        <div className={styles.footer}>
          <CollapseButton collapsed={collapsed} onToggle={toggleCollapsed} />
        </div>
      )}
    </div>
  );
}

interface NavItemsProps {
  items: NavItem[];
  collapsed: boolean;
  labelledBy?: string;
}

function NavItems({ items, collapsed, labelledBy }: NavItemsProps) {
  const { has } = useAccess();
  return (
    <ul className={styles.list} aria-labelledby={labelledBy}>
      {itemsFor(items, has).map((item) => {
        const entry = (
          <li key={item.path}>
            <NavItemLink item={item} collapsed={collapsed} />
          </li>
        );
        return item.Shown ? (
          <Suspense key={item.path} fallback={entry}>
            <item.Shown>{entry}</item.Shown>
          </Suspense>
        ) : (
          entry
        );
      })}
    </ul>
  );
}

/** Home is current only on Home; a section also on its sub-pages and its `alsoCurrentOn` addresses. */
function NavItemLink({ item, collapsed }: { item: NavItem; collapsed: boolean }) {
  const { pathname } = useLocation();
  const onSection = useMatch({ path: item.path, end: item.path === '/' }) !== null;
  const current =
    onSection || (item.alsoCurrentOn ?? []).some((start) => pathname.startsWith(start));
  // A plain class name: in the icon rail the link is a Tooltip trigger, which joins class names as text.
  const link = (
    <Link to={item.path} className={styles.item} aria-current={current ? 'page' : undefined}>
      <item.icon className={styles.icon} aria-hidden="true" />
      {/* Hidden on screen in the icon rail, never removed: the link keeps its name. */}
      <span className={collapsed ? 'visually-hidden' : styles.label}>{item.label}</span>
      {/* A count after the label, inside the link, so it is part of the link's name (feature 008). */}
      {item.Badge && (
        <Suspense fallback={null}>
          {' '}
          <item.Badge collapsed={collapsed} />
        </Suspense>
      )}
    </Link>
  );

  return collapsed ? (
    <Tooltip content={item.label} side="right">
      {link}
    </Tooltip>
  ) : (
    link
  );
}

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
          <ChevronsRight aria-hidden="true" />
        </button>
      </Tooltip>
    );
  }
  return (
    <button type="button" className={styles.collapse} onClick={onToggle}>
      <ChevronsLeft aria-hidden="true" />
      Collapse
    </button>
  );
}
