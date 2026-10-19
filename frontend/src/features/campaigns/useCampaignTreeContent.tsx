import type { ReactNode } from 'react';
import type { ActionsMenuItem, TreeNode, TreeTableFooter } from '@/components';

/** What feature 005 puts into feature 004's plan tree (005 plan, "Plan tree integration"). */
export interface CampaignTreeContent {
  /** One node per campaign with its initiatives as children: the plan node's children. */
  nodes: TreeNode[];
  /** **Add campaign**, for the tree's toolbar (PlanTree `toolbarActions`). */
  toolbarActions: ReactNode;
  /** **Add campaign**, for the plan row's ⋯ menu. */
  planRowActions: ActionsMenuItem[];
  /** **Add campaign**, under the empty tree's sentence. */
  emptyAction: ReactNode;
  /** The totals row ("Total · 2 campaigns · 4 initiatives · 7 activities"): the TreeTable footer. */
  footer: TreeTableFooter | undefined;
  /** The Move, Copy and Delete dialogs opened from rows; render once on the plan page. */
  dialogs: ReactNode;
  isLoading: boolean;
  error: unknown;
  refetch: () => void;
}

/**
 * The campaigns and initiatives of a plan for feature 004's plan page — STUB of the shared
 * foundation: no rows and no actions yet. Feature 005 replaces this file (it reads
 * GET /api/plans/{planId}/campaigns and adds the activity rows of `extensions.ts`); feature 004's
 * PlanPage already calls it, so neither feature changes the other's files.
 */
export function useCampaignTreeContent(planId: string): CampaignTreeContent {
  void planId;
  return {
    nodes: [],
    toolbarActions: null,
    planRowActions: [],
    emptyAction: null,
    footer: undefined,
    dialogs: null,
    isLoading: false,
    error: null,
    refetch: () => {},
  };
}
