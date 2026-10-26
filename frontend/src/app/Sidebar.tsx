import { ChevronsLeft, ChevronsRight } from 'lucide-react';
import { useId } from 'react';
import { Link, useLocation, useMatch } from 'react-router';
import { Logo, Tooltip } from '@/components';
import { useAccess } from '@/lib/access';
import { cx } from '@/lib/cx';
import { itemsFor, MAIN_ITEMS, MORE_ITEMS } from './navigation';
import type { NavItem } from './navigation';
import styles from './Sidebar.module.css';
import { useCurrentUser, useSession } from './useSession';

export interface SidebarProps {
  /**
   * Inside the menu drawer of narrow screens: always expanded, with the logo at the top and
   * without Collapse, so one Escape always closes the drawer.
   */
  inDrawer?: boolean;
}

/**
 * The left sidebar (FR-010, FR-011, design system §2): the sections, a "More" group, and Collapse
 * at the bottom. It is white, 240 px wide, with 40 px items; the current section has the 4 px
 * yellow "you are here" bar, bold text and the hover grey (Shell). Collapsed, it is a 64 px icon
 * rail; the person's choice is saved for them. The logo, bell and account menu sit in the top bar.
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
      {inDrawer && (
        <div className={styles.brand}>
          <Link to="/" className={styles.homeLink} aria-label="MAPP home">
            <Logo withWordmark />
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
  /** The id of the group's label, which names the list for screen readers. */
  labelledBy?: string;
}

/** One group of sidebar links: those the person's roles open (feature 014, FR-012). */
function NavItems({ items, collapsed, labelledBy }: NavItemsProps) {
  const { has } = useAccess();
  return (
    <ul className={styles.list} aria-labelledby={labelledBy}>
      {itemsFor(items, has).map((item) => (
        <li key={item.path}>
          <NavItemLink item={item} collapsed={collapsed} />
        </li>
      ))}
    </ul>
  );
}

/**
 * A sidebar link, marked with aria-current="page" on the current section, which also gives it its
 * highlight (Sidebar.module.css). Home is current only on Home itself; a section is current on its
 * sub-pages too, and on the addresses in its `alsoCurrentOn`.
 */
function NavItemLink({ item, collapsed }: { item: NavItem; collapsed: boolean }) {
  const { pathname } = useLocation();
  const onSection = useMatch({ path: item.path, end: item.path === '/' }) !== null;
  const current =
    onSection || (item.alsoCurrentOn ?? []).some((start) => pathname.startsWith(start));
  // A plain class name: in the icon rail the link is the trigger of a Tooltip, which joins class
  // names as text.
  const link = (
    <Link to={item.path} className={styles.item} aria-current={current ? 'page' : undefined}>
      <item.icon className={styles.icon} aria-hidden="true" />
      {/* Hidden on screen in the icon rail, never removed: the link keeps its name. */}
      <span className={collapsed ? 'visually-hidden' : styles.label}>{item.label}</span>
      {/* A count after the label, inside the link, so it is part of the link's name (feature 008). */}
      {item.Badge && (
        <>
          {' '}
          <item.Badge collapsed={collapsed} />
        </>
      )}
    </Link>
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

/** "Collapse" (chevrons and word), or in the icon rail the chevrons alone with "Expand sidebar". */
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
