/**
 * The sections of NorthStar: their names, addresses and sidebar icons (design system §2), and —
 * until a later feature builds a section — the sentence its "Coming soon" page shows (FR-015).
 * A feature that builds a section removes its sentence and gives it a real page in router.tsx.
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

/** A page people reach from the sidebar or the account menu. */
export interface NavItem {
  label: string;
  path: string;
  icon: LucideIcon;
}

/** A section a later feature builds; until then its page says in one sentence what it will do. */
export interface ComingSoonSection extends NavItem {
  comingSoon: string;
}

const HOME: NavItem = { label: 'Home', path: '/', icon: House };

const PLANS: ComingSoonSection = {
  label: 'Plans',
  path: '/plans',
  icon: ClipboardList,
  comingSoon:
    "Create yearly marketing plans, give them budgets and publish them when they're ready.",
};

const APPROVALS: ComingSoonSection = {
  label: 'Approvals',
  path: '/approvals',
  icon: CircleCheckBig,
  comingSoon: 'Approve or decline the activities waiting for you, and follow the ones you sent.',
};

const ALL_WORK: ComingSoonSection = {
  label: 'All work',
  path: '/all-work',
  icon: LayoutGrid,
  comingSoon:
    'See every campaign, initiative, activity and line item across plans as a grid, a calendar or a budget view.',
};

const SETTINGS: ComingSoonSection = {
  label: 'Settings',
  path: '/settings',
  icon: Settings,
  comingSoon:
    'Set up people and teams, models, activity types, Shell attributes, approval workflows, currencies and finance budget items.',
};

const RECYCLE_BIN: ComingSoonSection = {
  label: 'Recycle bin',
  path: '/recycle-bin',
  icon: Trash,
  comingSoon: 'Bring back anything that was deleted, together with everything that was under it.',
};

const ACTIVITY_LOG: ComingSoonSection = {
  label: 'Activity log',
  path: '/activity-log',
  icon: RotateCcwClock,
  comingSoon: 'See who changed what, and when, in every plan and setting.',
};

/** My profile: reached from the account menu, not from the sidebar. */
export const MY_PROFILE: ComingSoonSection = {
  label: 'My profile',
  path: '/profile',
  icon: UserRound,
  comingSoon: 'See your details and teams, and choose how dates and money are shown to you.',
};

/** The sidebar's main group, in order. */
export const MAIN_ITEMS: NavItem[] = [HOME, PLANS, APPROVALS, ALL_WORK];

/** The sidebar's MORE group, in order. */
export const MORE_ITEMS: NavItem[] = [SETTINGS, RECYCLE_BIN, ACTIVITY_LOG];

/** Every section that shows a "Coming soon" page for now. */
export const COMING_SOON_SECTIONS: ComingSoonSection[] = [
  PLANS,
  APPROVALS,
  ALL_WORK,
  SETTINGS,
  RECYCLE_BIN,
  ACTIVITY_LOG,
  MY_PROFILE,
];
