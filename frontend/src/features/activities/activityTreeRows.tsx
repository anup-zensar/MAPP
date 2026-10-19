import type { NavigateFunction } from 'react-router';
import type { ActionsMenuItem, TreeNode } from '@/components';

/** What an initiative row or panel tells feature 006 about itself. */
export interface InitiativeRow {
  id: string;
  name: string;
  planId: string;
}

/**
 * The activity rows of a plan's tree, by initiative id (006 plan, "Plan tree rows and Add
 * activity") — STUB of the shared foundation: no rows yet. Feature 006 replaces this file
 * (usePlanActivities → one TreeNode per activity); feature 005 calls it through extensions.ts.
 */
export function useActivityRows(planId: string): Map<string, TreeNode[]> {
  void planId;
  return new Map();
}

/**
 * The actions feature 006 adds to an initiative's ⋯ menu and its panel (**Add activity**, to
 * /plans/:planId/initiatives/:initiativeId/activities/new) — STUB: none yet.
 */
export function activityActionsForInitiative(
  initiative: InitiativeRow,
  navigate: NavigateFunction,
): ActionsMenuItem[] {
  void initiative;
  void navigate;
  return [];
}
