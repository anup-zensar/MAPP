/**
 * The sections of NorthStar: their names, addresses and sidebar icons (design system §2), and —
 * until a later feature builds a section — the sentence its "Coming soon" page shows (FR-015).
 * A feature that builds a section removes its sentence and gives it a real page in router.tsx.
 *
 * Plans (004–007), Global budgets (014), Approvals (008), All work (010), Financials (015),
 * Settings (002, 003), My profile (003), Recycle bin and Activity log (011) have their pages; no
 * section is "Coming soon" any more.
 */
import {
  CircleCheckBig,
  ClipboardList,
  House,
  Landmark,
  LayoutGrid,
  ReceiptText,
  RotateCcwClock,
  Settings,
  Trash,
  UserRound,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import type { ComponentType } from 'react';
import { ApprovalsNavBadge } from '@/features/approvals/ApprovalsNavBadge';
import type { Permission } from '@/lib/access';

/** A page people reach from the sidebar or the account menu. */
export interface NavItem {
  label: string;
  path: string;
  icon: LucideIcon;
  /**
   * A count shown inside the sidebar link, after its label (feature 008: Approvals "3 waiting for
   * you"); `collapsed` is the icon rail.
   */
  Badge?: ComponentType<{ collapsed: boolean }>;
  /**
   * Other address beginnings under which the section is current too: pages of the section whose
   * address does not start with its own (Plans: an activity's short link and line items, 006 and 007).
   */
  alsoCurrentOn?: string[];
  /**
   * What a role of the person must include for the link to be shown (feature 014, FR-012); left out
   * for sections everyone signed in uses. The page's address checks it too (RequirePermission).
   */
  permission?: Permission;
}

/** A section a later feature builds; until then its page says in one sentence what it will do. */
export interface ComingSoonSection extends NavItem {
  comingSoon: string;
}

const HOME: NavItem = { label: 'Home', path: '/', icon: House };

const PLANS: NavItem = {
  label: 'Plans',
  path: '/plans',
  icon: ClipboardList,
  alsoCurrentOn: ['/activities/', '/line-items/'],
};

/** Feature 014 US6: global budgets shared out to market plans (FR-036). */
const GLOBAL_BUDGETS: NavItem = {
  label: 'Global budgets',
  path: '/budgets',
  icon: Landmark,
  permission: 'globalBudgets.view',
};

const APPROVALS: NavItem = {
  label: 'Approvals',
  path: '/approvals',
  icon: CircleCheckBig,
  Badge: ApprovalsNavBadge,
};

const ALL_WORK: NavItem = { label: 'All work', path: '/all-work', icon: LayoutGrid };

/** Feature 015: purchase orders, invoices, payments and accruals from GSAP (FR-020). */
const FINANCIALS: NavItem = {
  label: 'Financials',
  path: '/financials',
  icon: ReceiptText,
  permission: 'finance.view',
};

const SETTINGS: NavItem = { label: 'Settings', path: '/settings', icon: Settings };

const RECYCLE_BIN: NavItem = { label: 'Recycle bin', path: '/recycle-bin', icon: Trash };

const ACTIVITY_LOG: NavItem = {
  label: 'Activity log',
  path: '/activity-log',
  icon: RotateCcwClock,
};

/** My profile: reached from the account menu, not from the sidebar. */
export const MY_PROFILE: NavItem = { label: 'My profile', path: '/profile', icon: UserRound };

/** The sidebar's main group, in order. */
export const MAIN_ITEMS: NavItem[] = [HOME, PLANS, GLOBAL_BUDGETS, APPROVALS, ALL_WORK, FINANCIALS];

/** The sidebar's MORE group, in order. */
export const MORE_ITEMS: NavItem[] = [SETTINGS, RECYCLE_BIN, ACTIVITY_LOG];

/** Every section that shows a "Coming soon" page for now. */
export const COMING_SOON_SECTIONS: ComingSoonSection[] = [];

/**
 * The items the person's roles open (feature 014, FR-012): items without a permission, and those
 * whose permission `has` grants.
 */
export function itemsFor<T extends { permission?: Permission }>(
  items: T[],
  has: (permission: Permission) => boolean,
): T[] {
  return items.filter((item) => !item.permission || has(item.permission));
}
