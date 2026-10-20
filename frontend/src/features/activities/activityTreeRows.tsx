import { useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { useNavigate } from 'react-router';
import type { NavigateFunction } from 'react-router';
import { ApiError } from '@/api/errors';
import { PersonLabel, useToast } from '@/components';
import type { ActionsMenuItem, TreeNode } from '@/components';
import type { components } from '@/api/schema';
import { gateItem } from '@/features/plans/gateAction';
import { useFormatters } from '@/lib/useFormatters';
import { fetchLatestActivity, usePlanActivities } from './activitiesApi';
import type { Activity, ActivitySummary } from './activitiesApi';
import { ActivityDeleteQuestion } from './ActivityDeleteQuestion';
import { ActivityRowCost, ActivityRowStatus } from './ActivityRowCells';
import { activityEditHref, activityHref, datesText, EMPTY, newActivityHref } from './activityText';
import type { DateWriters } from './activityText';
import { CopyActivityDialog } from './CopyActivityDialog';
import { MoveActivityDialog } from './MoveActivityDialog';
import { SplitActivityDialog } from './SplitActivityDialog';
import { SPLIT_ACTIVITY } from './splitText';

type PlanContentAccess = components['schemas']['PlanContentAccess'];

/** What an initiative row or panel tells feature 006 about itself. */
export interface InitiativeRow {
  id: string;
  name: string;
  planId: string;
  /** Whether the person may add activities to it (its access's addChild, feature 014). */
  addActivity?: components['schemas']['ActionAvailability'];
}

/** The dialogs a row's ⋯ menu opens (FR-029 – FR-031). */
export type RowDialog = 'copy' | 'move' | 'split' | 'delete';

/**
 * Copy, Move, Split activity and Delete of the rows' ⋯ menus: the open dialog, and how to open and
 * close it.
 */
export interface RowActions {
  /** The open dialog and its activity as stored now; null when none is open. */
  open: { dialog: RowDialog; activity: Activity } | null;
  ask: (dialog: RowDialog, activityId: string) => void;
  close: () => void;
}

/**
 * The activity rows of a plan's tree, by initiative id (006 plan, "Plan tree rows and Add
 * activity", FR-033); feature 005 places them under its initiative rows through extensions.ts.
 * Each row: the level bar and tag, the name as a link to the activity page with the ID beside it,
 * the owner, the dates, the status and completeness chips, the planned cost (marked red when line
 * items exceed it) and the ⋯ menu: View, Edit, Copy, Move (FR-029, FR-030) and Delete (FR-031),
 * which asks first and offers Undo.
 *
 * The cells use the plan tree's column keys: name (the ID), owner, dates, status, plannedCost.
 */
export function useActivityRows(planId: string): Map<string, TreeNode[]> {
  const query = usePlanActivities(planId);
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { show } = useToast();
  const format = useFormatters();
  const [open, setOpen] = useState<RowActions['open']>(null);

  async function ask(dialog: RowDialog, activityId: string) {
    // The dialogs need the whole activity as stored now: its line items (delete, copy) and its
    // version (move).
    try {
      setOpen({ dialog, activity: await fetchLatestActivity(queryClient, activityId) });
    } catch (error) {
      show({ message: (error instanceof ApiError ? error : new ApiError('other')).message });
    }
  }

  return activityRowsByInitiative(
    query.data?.items ?? [],
    planId,
    navigate,
    format,
    {
      open,
      ask: (dialog, activityId) => void ask(dialog, activityId),
      close: () => setOpen(null),
    },
    query.data?.access,
  );
}

/** One TreeNode per activity, grouped by initiative id, in the API's order. */
export function activityRowsByInitiative(
  activities: ActivitySummary[],
  planId: string,
  navigate: NavigateFunction,
  format: DateWriters,
  rowActions?: RowActions,
  /** What the person may do with the plan's activities (feature 014); missing means everything. */
  access?: PlanContentAccess,
): Map<string, TreeNode[]> {
  const rows = new Map<string, TreeNode[]>();
  for (const activity of activities) {
    const href = activityHref(planId, activity.id);
    const actions: ActionsMenuItem[] = [
      { label: 'View', onSelect: () => void navigate(href) },
      ...gateItem(
        { label: 'Edit', onSelect: () => void navigate(activityEditHref(planId, activity.id)) },
        access?.editActivities,
      ),
    ];
    if (rowActions) {
      actions.push(
        ...gateItem(
          { label: 'Copy', onSelect: () => rowActions.ask('copy', activity.id) },
          access?.editActivities,
        ),
        ...gateItem(
          { label: 'Move', onSelect: () => rowActions.ask('move', activity.id) },
          access?.editActivities,
        ),
        // Split off line items and planned cost into a new activity (016 FR-026).
        ...gateItem(
          { label: SPLIT_ACTIVITY, onSelect: () => rowActions.ask('split', activity.id) },
          access?.editActivities,
        ),
        ...gateItem(
          {
            label: 'Delete',
            tone: 'danger',
            separatorBefore: true,
            onSelect: () => rowActions.ask('delete', activity.id),
          },
          access?.deleteActivities,
        ),
      );
    }
    const opened = rowActions?.open?.activity.id === activity.id ? rowActions.open : null;
    const node: TreeNode = {
      id: activity.id,
      level: 'activity',
      name: activity.name,
      href,
      cells: {
        name: (
          <>
            {activity.id}
            {/* The dialogs are drawn over the page, from the row they are about. */}
            {opened && rowActions && opened.dialog === 'delete' && (
              <ActivityDeleteQuestion
                activity={opened.activity}
                open
                onOpenChange={(open) => {
                  if (!open) {
                    rowActions.close();
                  }
                }}
              />
            )}
            {opened && rowActions && opened.dialog === 'copy' && (
              <CopyActivityDialog activity={opened.activity} onClose={rowActions.close} />
            )}
            {opened && rowActions && opened.dialog === 'move' && (
              <MoveActivityDialog activity={opened.activity} onClose={rowActions.close} />
            )}
            {opened && rowActions && opened.dialog === 'split' && (
              <SplitActivityDialog activity={opened.activity} onClose={rowActions.close} />
            )}
          </>
        ),
        owner: activity.owner ? <PersonLabel person={activity.owner} /> : EMPTY,
        dates: datesText(activity.startDate, activity.endDate, activity.ongoing, format),
        status: <ActivityRowStatus row={activity} href={href} />,
        plannedCost: <ActivityRowCost row={activity} />,
      },
      children: [],
      actions,
    };
    rows.set(activity.initiativeId, [...(rows.get(activity.initiativeId) ?? []), node]);
  }
  return rows;
}

/**
 * The actions feature 006 adds to an initiative's ⋯ menu and its panel's More menu (FR-001):
 * **Add activity**, which opens the new-activity page of the initiative.
 */
export function activityActionsForInitiative(
  initiative: InitiativeRow,
  navigate: NavigateFunction,
): ActionsMenuItem[] {
  return gateItem(
    {
      label: 'Add activity',
      onSelect: () => void navigate(newActivityHref(initiative.planId, initiative.id)),
    },
    initiative.addActivity,
  );
}
