import { useState } from 'react';
import { useNavigate } from 'react-router';
import type { components } from '@/api/schema';
import { ApiError } from '@/api/errors';
import { useToast } from '@/components';
import type { ActionsMenuItem } from '@/components';
import { useUndoableDelete } from '@/lib/useUndoableDelete';
import { DeletePlanDialog } from './dialogs/DeletePlanDialog';
import { PublishBlockedDialog } from './dialogs/PublishBlockedDialog';
import { UnpublishDialog } from './dialogs/UnpublishDialog';
import { PLAN_KEYS, useDeletePlan, usePublishPlan, useUnpublishPlan } from './plansApi';
import type { PlanStatus } from './planText';

type ReadinessGroup = components['schemas']['ReadinessGroup'];

/** What the actions need to know of a plan (a list row or the plan itself). */
export interface ActionPlan {
  id: string;
  name: string;
  status: PlanStatus;
}

export interface PlanActionsOptions {
  /** After a delete: the plan page goes to the Plans list; the list stays. */
  onDeleted?: () => void;
}

/**
 * The actions on a plan, shared by the Plans list and the plan page (plan "Dialogs"): View, Edit,
 * Publish or Unpublish, and Delete with Undo, with their dialogs. Render `dialogs` once on the page.
 */
export function usePlanActions({ onDeleted }: PlanActionsOptions = {}) {
  const navigate = useNavigate();
  const { show } = useToast();
  const publishPlan = usePublishPlan();
  const unpublishPlan = useUnpublishPlan();
  const deletePlan = useDeletePlan();
  const afterDelete = useUndoableDelete({ invalidate: [PLAN_KEYS.all] });
  const [deleting, setDeleting] = useState<{ plan: ActionPlan; open: boolean } | null>(null);
  const [blocked, setBlocked] = useState<{
    planId: string;
    missing: ReadinessGroup[];
    open: boolean;
  } | null>(null);
  const [unpublishing, setUnpublishing] = useState<{ plan: ActionPlan; open: boolean } | null>(
    null,
  );
  // A new key each time the unpublish dialog opens, so it starts empty.
  const [unpublishKey, setUnpublishKey] = useState(0);

  /** Publishes a Draft plan, or lists every field it still needs (FR-024). */
  async function publish(plan: ActionPlan) {
    if (publishPlan.isPending) {
      return;
    }
    try {
      const published = await publishPlan.mutateAsync(plan.id);
      show({ message: `Plan '${published.name}' published` });
    } catch (error) {
      const apiError = error instanceof ApiError ? error : new ApiError('other');
      if (apiError.kind === 'notReady') {
        setBlocked({ planId: plan.id, missing: apiError.missing, open: true });
      } else {
        show({ message: apiError.message });
      }
    }
  }

  /** Asks why, then returns a Published plan to Draft (FR-025). */
  function unpublish(plan: ActionPlan) {
    setUnpublishKey((key) => key + 1);
    setUnpublishing({ plan, open: true });
  }

  async function confirmUnpublish(plan: ActionPlan, reason: string): Promise<string | null> {
    try {
      const draft = await unpublishPlan.mutateAsync({ planId: plan.id, reason });
      setUnpublishing({ plan, open: false });
      show({ message: `Plan '${draft.name}' unpublished — it's a Draft again` });
      return null;
    } catch (error) {
      const apiError = error instanceof ApiError ? error : new ApiError('other');
      const problem = apiError.fields.find((field) => field.field === 'reason');
      if (problem) {
        return problem.message;
      }
      setUnpublishing({ plan, open: false });
      show({ message: apiError.message });
      return null;
    }
  }

  /** Asks first, then moves the plan and everything in it to the recycle bin (FR-029). */
  function remove(plan: ActionPlan) {
    setDeleting({ plan, open: true });
  }

  async function confirmDelete(plan: ActionPlan) {
    try {
      const result = await deletePlan.mutateAsync(plan.id);
      setDeleting({ plan, open: false });
      // "Plan '<name>' moved to the recycle bin" with Undo for 10 seconds (FR-030).
      afterDelete(result);
      onDeleted?.();
    } catch (error) {
      setDeleting({ plan, open: false });
      show({ message: (error instanceof ApiError ? error : new ApiError('other')).message });
    }
  }

  /** The ⋯ Delete item: last, after a line, in red (design system §6.1). */
  function deleteItem(plan: ActionPlan): ActionsMenuItem {
    return { label: 'Delete', tone: 'danger', separatorBefore: true, onSelect: () => remove(plan) };
  }

  /** The list's ⋯ Actions menu: View, Edit, Publish or Unpublish, Delete (FR-001). */
  function menuItems(plan: ActionPlan): ActionsMenuItem[] {
    const path = `/plans/${encodeURIComponent(plan.id)}`;
    return [
      { label: 'View', onSelect: () => void navigate(path) },
      { label: 'Edit', onSelect: () => void navigate(`${path}/edit`) },
      plan.status === 'draft'
        ? { label: 'Publish', onSelect: () => void publish(plan) }
        : { label: 'Unpublish', onSelect: () => unpublish(plan) },
      deleteItem(plan),
    ];
  }

  const dialogs = (
    <>
      <PublishBlockedDialog
        open={blocked?.open ?? false}
        planId={blocked?.planId ?? ''}
        missing={blocked?.missing ?? []}
        onClose={() => blocked && setBlocked({ ...blocked, open: false })}
      />
      {unpublishing && (
        <UnpublishDialog
          key={unpublishKey}
          open={unpublishing.open}
          planName={unpublishing.plan.name}
          onUnpublish={(reason) => confirmUnpublish(unpublishing.plan, reason)}
          onClose={() => setUnpublishing({ ...unpublishing, open: false })}
        />
      )}
      {deleting && (
        <DeletePlanDialog
          open={deleting.open}
          planId={deleting.plan.id}
          planName={deleting.plan.name}
          busy={deletePlan.isPending}
          onDelete={() => void confirmDelete(deleting.plan)}
          onClose={() => setDeleting({ ...deleting, open: false })}
        />
      )}
    </>
  );

  return { menuItems, deleteItem, publish, unpublish, remove, dialogs };
}
