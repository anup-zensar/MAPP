import type { ActionsMenuItem } from '@/components';
import type { ApprovalWorkflow } from './workflowsApi';

/** What the list and the details panel let people do with a workflow. */
export interface WorkflowActions {
  edit: (workflow: ApprovalWorkflow) => void;
  setInactive: (workflow: ApprovalWorkflow) => void;
  setActive: (workflow: ApprovalWorkflow) => void;
  remove: (workflow: ApprovalWorkflow) => void;
}

/** Set inactive or Reactivate · Delete (last, after a line), in the row's and panel's ⋯. */
export function workflowMoreItems(
  workflow: ApprovalWorkflow,
  actions: WorkflowActions,
): ActionsMenuItem[] {
  return [
    workflow.active
      ? { label: 'Set inactive', onSelect: () => actions.setInactive(workflow) }
      : { label: 'Reactivate', onSelect: () => actions.setActive(workflow) },
    {
      label: 'Delete',
      tone: 'danger',
      separatorBefore: true,
      onSelect: () => actions.remove(workflow),
    },
  ];
}
