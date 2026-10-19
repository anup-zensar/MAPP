/**
 * The sections of NorthStar: their names, addresses and sidebar icons (design system §2), and —
 * until a later feature builds a section — the sentence its "Coming soon" page shows (FR-015).
 * A feature that builds a section removes its sentence and gives it a real page in router.tsx.
 *
 * Plans (004), Approvals (008), Settings (002) and My profile (003) have their pages (stubs until
 * those features replace them); so do Recycle bin and Activity log (011). All work (010) comes later.
 */
import {
  CircleCheckBig,
  ClipboardList,
  House,
  LayoutGrid,
  RotateCcwClock,
  Settings,
  Trash,
  UserRound,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import type { ComponentType } from 'react';
import { ApprovalsNavBadge } from '@/features/approvals/ApprovalsNavBadge';

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
}

/** A section a later feature builds; until then its page says in one sentence what it will do. */
export interface ComingSoonSection extends NavItem {
  comingSoon: string;
}

const HOME: NavItem = { label: 'Home', path: '/', icon: House };

const PLANS: NavItem = { label: 'Plans', path: '/plans', icon: ClipboardList };

const APPROVALS: NavItem = {
  label: 'Approvals',
  path: '/approvals',
  icon: CircleCheckBig,
  Badge: ApprovalsNavBadge,
};

const ALL_WORK: ComingSoonSection = {
  label: 'All work',
  path: '/all-work',
  icon: LayoutGrid,
  comingSoon:
    'See every campaign, initiative, activity and line item across plans as a grid, a calendar or a budget view.',
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
export const MAIN_ITEMS: NavItem[] = [HOME, PLANS, APPROVALS, ALL_WORK];

/** The sidebar's MORE group, in order. */
export const MORE_ITEMS: NavItem[] = [SETTINGS, RECYCLE_BIN, ACTIVITY_LOG];

/** Every section that shows a "Coming soon" page for now. */
export const COMING_SOON_SECTIONS: ComingSoonSection[] = [ALL_WORK];
