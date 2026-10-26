import { Plus } from 'lucide-react';
import type { ReactNode } from 'react';
import { useLocation, useNavigate } from 'react-router';
import { LinkButton } from '@/ui/Button/Button';
import type { ActionsMenuItem } from '@/ui/ActionsMenu/ActionsMenu';
import type { TreeNode, TreeTableFooter } from '@/ui/TreeTable/TreeTable';
import { allows, gateItem } from '@/features/plans/gateAction';
import type { Availability } from '@/lib/access';
import { useFormatters } from '@/lib/useFormatters';
import { usePlanCampaigns } from './campaignApi';
import { campaignRows, totalsFooter } from './campaignRows';
import { useRowsBelowInitiatives } from './extensions';
import { ADD_CAMPAIGN } from './texts';
import { useRecordActions } from './useRecordActions';

/** What feature 005 puts into feature 004's plan tree (005 plan, "Plan tree integration"). */
export interface CampaignTreeContent {
  /**
   * One node per campaign with its initiatives as children: the plan node's children. Cells by
   * column key: name, owner, dates, status, budget, plannedCost, budgetUsed (campaignRows.tsx).
   * Plan and campaign rows start expanded (FR-020); the highlighted row's ancestors open too.
   */
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
 * The campaigns and initiatives of a plan for feature 004's plan page (FR-001, FR-012, FR-019 –
 * FR-022): GET /api/v1/plans/{planId}/campaigns as tree rows with feature 006's activity rows under
 * the initiatives, **Add campaign** for the toolbar, the plan row and the empty tree, and the
 * totals row. Every change refreshes the list (refreshPlanning), so the rows follow at once.
 */
export function useCampaignTreeContent(
  planId: string,
  /** Whether the person may add campaigns to the plan (the plan's access, feature 014). */
  addCampaignAccess?: Availability,
): CampaignTreeContent {
  const query = usePlanCampaigns(planId);
  const navigate = useNavigate();
  const location = useLocation();
  const formatters = useFormatters();
  const rowsBelow = useRowsBelowInitiatives(planId);
  const actions = useRecordActions();
  const newHref = `/plans/${planId}/campaigns/new`;
  const addCampaign = allows(addCampaignAccess) ? (
    <LinkButton to={newHref} icon={<Plus />}>
      {ADD_CAMPAIGN}
    </LinkButton>
  ) : undefined;

  return {
    nodes: query.data
      ? campaignRows(query.data, {
          planId,
          formatters,
          navigate,
          highlightId: highlightOf(location.state),
          rowsBelow,
          onDelete: actions.askDelete,
          onCopy: actions.askCopy,
          onMove: actions.askMove,
        })
      : [],
    toolbarActions: addCampaign,
    planRowActions: gateItem(
      { label: ADD_CAMPAIGN, onSelect: () => void navigate(newHref) },
      addCampaignAccess,
    ),
    emptyAction: addCampaign,
    footer: query.data ? totalsFooter(query.data, formatters) : undefined,
    dialogs: actions.dialogs,
    isLoading: query.isPending,
    error: query.error,
    refetch: () => void query.refetch(),
  };
}

/** The record to highlight, from the location state a create, move or copy leaves (FR-004). */
function highlightOf(state: unknown): string | undefined {
  if (state && typeof state === 'object' && 'highlight' in state) {
    const { highlight } = state;
    return typeof highlight === 'string' ? highlight : undefined;
  }
  return undefined;
}
