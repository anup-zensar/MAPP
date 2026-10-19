import { useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { useNavigate } from 'react-router';
import type { NavigateFunction } from 'react-router';
import { ApiError } from '@/api/errors';
import { PersonLabel, useToast } from '@/components';
import type { ActionsMenuItem, TreeNode } from '@/components';
import { useFormatters } from '@/lib/useFormatters';
import { fetchLatestActivity, usePlanActivities } from './activitiesApi';
import type { Activity, ActivitySummary } from './activitiesApi';
import { ActivityDeleteQuestion } from './ActivityDeleteQuestion';
import { ActivityRowCost, ActivityRowStatus } from './ActivityRowCells';
import { activityEditHref, activityHref, datesText, EMPTY, newActivityHref } from './activityText';

/** What an initiative row or panel tells feature 006 about itself. */
export interface InitiativeRow {
  id: string;
  name: string;
  planId: string;
}

/** The Delete of the rows' ⋯ menus: the activity being asked about, and how to ask and stop. */
export interface RowDelete {
  /** The activity whose delete question is open, as stored now; null when none is. */
  activity: Activity | null;
  ask: (activityId: string) => void;
  close: () => void;
}

/**
 * The activity rows of a plan's tree, by initiative id (006 plan, "Plan tree rows and Add
 * activity", FR-033); feature 005 places them under its initiative rows through extensions.ts.
 * Each row: the level bar and tag, the name as a link to the activity page with the ID beside it,
 * the owner, the dates, the status and completeness chips, the planned cost (marked red when line
 * items exceed it) and the ⋯ menu: View, Edit and Delete (FR-031), which asks first and offers
 * Undo.
 *
 * The cells use the plan tree's column keys: name (the ID), owner, dates, status, plannedCost.
 */
export function useActivityRows(planId: string): Map<string, TreeNode[]> {
  const query = usePlanActivities(planId);
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { show } = useToast();
  const { date } = useFormatters();
  const [deleting, setDeleting] = useState<Activity | null>(null);

  async function ask(activityId: string) {
    // The question says how many line items go with it, which only the whole activity tells.
    try {
      setDeleting(await fetchLatestActivity(queryClient, activityId));
    } catch (error) {
      show({ message: (error instanceof ApiError ? error : new ApiError('other')).message });
    }
  }

  return activityRowsByInitiative(query.data?.items ?? [], planId, navigate, date, {
    activity: deleting,
    ask: (activityId) => void ask(activityId),
    close: () => setDeleting(null),
  });
}

/** One TreeNode per activity, grouped by initiative id, in the API's order. */
export function activityRowsByInitiative(
  activities: ActivitySummary[],
  planId: string,
  navigate: NavigateFunction,
  date: (date: string) => string,
  remove?: RowDelete,
): Map<string, TreeNode[]> {
  const rows = new Map<string, TreeNode[]>();
  for (const activity of activities) {
    const href = activityHref(planId, activity.id);
    const actions: ActionsMenuItem[] = [
      { label: 'View', onSelect: () => void navigate(href) },
      { label: 'Edit', onSelect: () => void navigate(activityEditHref(planId, activity.id)) },
    ];
    if (remove) {
      actions.push({
        label: 'Delete',
        tone: 'danger',
        separatorBefore: true,
        onSelect: () => remove.ask(activity.id),
      });
    }
    const question = remove?.activity?.id === activity.id ? remove.activity : null;
    const node: TreeNode = {
      id: activity.id,
      level: 'activity',
      name: activity.name,
      href,
      cells: {
        name: (
          <>
            {activity.id}
            {question && remove && (
              // The question is drawn over the page, from the row it asks about.
              <ActivityDeleteQuestion
                activity={question}
                open
                onOpenChange={(open) => {
                  if (!open) {
                    remove.close();
                  }
                }}
              />
            )}
          </>
        ),
        owner: activity.owner ? <PersonLabel person={activity.owner} /> : EMPTY,
        dates: datesText(activity.startDate, activity.endDate, activity.ongoing, date),
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
  return [
    {
      label: 'Add activity',
      onSelect: () => void navigate(newActivityHref(initiative.planId, initiative.id)),
    },
  ];
}
