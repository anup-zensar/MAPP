import { useState } from 'react';
import { useNavigate } from 'react-router';
import type { components } from '@/api/generated/schema';
import { ApiError } from '@/api/errors';
import { useToast } from '@/ui/Toast/useToast';
import type { ActionsMenuItem } from '@/ui/ActionsMenu/ActionsMenu';
import { useUndoableDelete } from '@/ui/useUndoableDelete';
import { gateItem } from '@/session/gate-action';
import { CopyPlanDialog } from './dialogs/CopyPlanDialog';
import type { CopyRefusal } from './dialogs/CopyPlanDialog';
import { DeletePlanDialog } from './dialogs/DeletePlanDialog';
import { PublishBlockedDialog } from './dialogs/PublishBlockedDialog';
import { UnpublishDialog } from './dialogs/UnpublishDialog';
import { PLAN_KEYS, useCopyPlan, useDeletePlan, usePublishPlan, useUnpublishPlan } from './api';
import type { PlanCopyRequest } from './api';
import type { PlanStatus } from './plan-text';

type ReadinessGroup = components['schemas']['ReadinessGroup'];
type PlanAccess = components['schemas']['PlanAccess'];

/** What the actions need to know of a plan (a list row or the plan itself). */
export interface ActionPlan {
  id: string;
  name: string;
  status: PlanStatus;
  /** What the signed-in person may do with the plan (feature 014); missing means everything. */
  access?: PlanAccess;
}

export interface PlanActionsOptions {
  /** After a delete: the plan page goes to the Plans list; the list stays. */
  onDeleted?: () => void;
}

/**
 * The actions on a plan, shared by the Plans list and the plan page (plan "Dialogs"): View, Edit,
 * Copy, Publish or Unpublish, and Delete with Undo, with their dialogs. Render `dialogs` once on
 * the page.
 */
export function usePlanActions({ onDeleted }: PlanActionsOptions = {}) {
  const navigate = useNavigate();
  const { show } = useToast();
  const publishPlan = usePublishPlan();
  const unpublishPlan = useUnpublishPlan();
  const copyPlan = useCopyPlan();
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
  const [copying, setCopying] = useState<{ plan: ActionPlan; open: boolean } | null>(null);
  // A new key each time the copy dialog opens, so it starts from the API's suggestion.
  const [copyKey, setCopyKey] = useState(0);

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

  /** Asks for the name, the year and the contents, then copies the plan (FR-026). */
  function copy(plan: ActionPlan) {
    setCopyKey((key) => key + 1);
    setCopying({ plan, open: true });
  }

  /**
   * Makes the copy; on success its plan page opens with the message and one message per note, such
   * as a cleared default finance budget (FR-027, FR-028).
   */
  async function confirmCopy(
    plan: ActionPlan,
    request: PlanCopyRequest,
  ): Promise<CopyRefusal | null> {
    try {
      const { plan: copied, notes } = await copyPlan.mutateAsync({
        planId: plan.id,
        body: request,
      });
      setCopying({ plan, open: false });
      void navigate(`/plans/${encodeURIComponent(copied.id)}`);
      show({ message: `Plan '${copied.name}' created as a copy of '${plan.name}'` });
      notes.forEach((note) => show({ message: note }));
      return null;
    } catch (error) {
      const apiError = error instanceof ApiError ? error : new ApiError('other');
      if (apiError.fields.length > 0) {
        return { problems: apiError.fields };
      }
      if (apiError.kind === 'notFound') {
        // The plan was deleted meanwhile: the message, and the refreshed list without it.
        setCopying({ plan, open: false });
        show({ message: apiError.message });
        return null;
      }
      return { problems: [], message: apiError.message };
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
      // "Plan '<name>' deleted" with Undo for 10 seconds (FR-030).
      afterDelete(result);
      onDeleted?.();
    } catch (error) {
      setDeleting({ plan, open: false });
      show({ message: (error instanceof ApiError ? error : new ApiError('other')).message });
    }
  }

  /**
   * The ⋯ Delete item: last, after a line, in red (design system §6.1). Disabled with the reason, or
   * left out, when the person may not delete the plan (014 FR-011): spread it into the menu.
   */
  function deleteItem(plan: ActionPlan): ActionsMenuItem[] {
    return gateItem(
      { label: 'Delete', tone: 'danger', separatorBefore: true, onSelect: () => remove(plan) },
      plan.access?.delete,
    );
  }

  /** The ⋯ Copy item, on the list and the plan page (FR-026), as the person's access allows it. */
  function copyItem(plan: ActionPlan): ActionsMenuItem[] {
    return gateItem({ label: 'Copy', onSelect: () => copy(plan) }, plan.access?.copy);
  }

  /** The list's ⋯ Actions menu: View, Edit, Copy, Publish or Unpublish, Delete (FR-001). */
  function menuItems(plan: ActionPlan): ActionsMenuItem[] {
    const path = `/plans/${encodeURIComponent(plan.id)}`;
    return [
      { label: 'View', onSelect: () => void navigate(path) },
      ...gateItem(
        { label: 'Edit', onSelect: () => void navigate(`${path}/edit`) },
        plan.access?.edit,
      ),
      ...copyItem(plan),
      ...gateItem(
        plan.status === 'draft'
          ? { label: 'Publish', onSelect: () => void publish(plan) }
          : { label: 'Unpublish', onSelect: () => unpublish(plan) },
        plan.access?.publish,
      ),
      ...deleteItem(plan),
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
      {copying && (
        <CopyPlanDialog
          key={copyKey}
          open={copying.open}
          planId={copying.plan.id}
          planName={copying.plan.name}
          onCopy={(request) => confirmCopy(copying.plan, request)}
          onClose={() => setCopying({ ...copying, open: false })}
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

  return { menuItems, copyItem, deleteItem, publish, unpublish, copy, remove, dialogs };
}
