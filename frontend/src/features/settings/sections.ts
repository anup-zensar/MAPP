/**
 * Feature 002's cards of the Settings page (FR-001). The page's full list, with the cards other
 * features add from their own folders, is assembled in app/settings-sections.ts.
 */
import { Calculator, Coins, ListChecks, Shapes, Wallet } from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import type { ComponentType } from 'react';
import type { components } from '@/api/generated/schema';
import type { Permission } from '@/session/access';

/** One card of the Settings page. */
export interface SettingsSection {
  title: string;
  /** One sentence on what the section holds (plan 002, "Settings cards"). */
  description: string;
  path: string;
  icon: LucideIcon;
  /** 002's sections: the count of active records in GET /api/v1/settings/overview. */
  overviewKey?: keyof components['schemas']['SettingsOverview'];
  /**
   * Other features' sections: a component that shows "<n> active" from their own data. Left out
   * until the feature builds it.
   */
  ActiveCount?: ComponentType;
  /**
   * A section that has no page yet: its card shows "Coming soon" and is not a link (FR-001). Left
   * out for every section with a page — a page still being built is a link to its stub.
   */
  comingSoon?: boolean;
  /**
   * What a role of the person must include for the card to be shown (feature 014, FR-012), such as
   * "access.manage" for Roles and access; left out for the cards everyone reads (D-240).
   */
  permission?: Permission;
}

/** Feature 002's own sections. */
export const SETTINGS_OWN_SECTIONS: SettingsSection[] = [
  {
    title: 'Models',
    description:
      'The rulebooks plans use: conversion rates, average deal size and allowed activity types.',
    path: '/settings/models',
    icon: Calculator,
    overviewKey: 'models',
  },
  {
    title: 'Activity types',
    description:
      'The kinds of work an activity can be, each with an optional default planned cost.',
    path: '/settings/activity-types',
    icon: Shapes,
    overviewKey: 'activityTypes',
  },
  {
    title: 'Shell attributes',
    description: "Shell's own fields on each level's form, with their options and markings.",
    path: '/settings/shell-attributes',
    icon: ListChecks,
    overviewKey: 'shellAttributes',
  },
  {
    title: 'Currencies',
    description:
      'The currencies line items, plans and people can use, with their rate history and GSAP rate import.',
    path: '/settings/currencies',
    icon: Coins,
    overviewKey: 'currencies',
  },
  {
    title: 'Finance budget items',
    description:
      'The finance budget items line items link to — a simple list until the Finance module arrives.',
    path: '/settings/finance-budget-items',
    icon: Wallet,
    overviewKey: 'financeBudgetItems',
  },
];
