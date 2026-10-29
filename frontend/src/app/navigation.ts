import {
  CircleCheckBig,
  ClipboardList,
  FileText,
  House,
  Landmark,
  LayoutDashboard,
  LayoutGrid,
  ReceiptText,
  RotateCcwClock,
  Settings,
  Trash,
  UserRound,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import { lazy } from 'react';
import type { ComponentType } from 'react';
import type { Permission } from '@/session/access';

// The badge loads with the approvals code after the frame is drawn, off the first page.
const ApprovalsNavBadge = lazy(() =>
  import('@/features/approvals').then((feature) => ({ default: feature.ApprovalsNavBadge })),
);

export interface NavItem {
  label: string;
  path: string;
  icon: LucideIcon;
  Badge?: ComponentType<{ collapsed: boolean }>;
  /** Other address beginnings under which the section is current too (Plans: line items). */
  alsoCurrentOn?: string[];
  /** Shown only to people who hold it; the address checks it too (RequirePermission). */
  permission?: Permission;
}

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

const DASHBOARDS: NavItem = {
  label: 'Dashboards',
  path: '/dashboards',
  icon: LayoutDashboard,
  permission: 'plans.view',
};

const FINANCIALS: NavItem = {
  label: 'Financials',
  path: '/financials',
  icon: ReceiptText,
  permission: 'finance.view',
};

const PURCHASE_REQUESTS: NavItem = {
  label: 'Purchase requests',
  path: '/purchase-requests',
  icon: FileText,
  permission: 'purchaseRequests.list',
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

export const MAIN_ITEMS: NavItem[] = [
  HOME,
  PLANS,
  GLOBAL_BUDGETS,
  APPROVALS,
  ALL_WORK,
  FINANCIALS,
  PURCHASE_REQUESTS,
  DASHBOARDS,
];

export const MORE_ITEMS: NavItem[] = [SETTINGS, RECYCLE_BIN, ACTIVITY_LOG];

export const COMING_SOON_SECTIONS: ComingSoonSection[] = [];
