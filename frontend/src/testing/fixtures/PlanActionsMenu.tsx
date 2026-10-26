import { usePlanActions } from '@/features/plans/usePlanActions';
import type { ActionPlan } from '@/features/plans/usePlanActions';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';

/** The Plans list's ⋯ menu of one plan, with the dialogs its actions open. */
export function PlanActionsMenu({ target }: { target: ActionPlan }) {
  const actions = usePlanActions();
  return (
    <>
      <ActionsMenu label={target.name} items={actions.menuItems(target)} />
      {actions.dialogs}
    </>
  );
}
