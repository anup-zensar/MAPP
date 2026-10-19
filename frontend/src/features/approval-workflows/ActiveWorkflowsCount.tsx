import { useWorkflows } from './workflowsApi';

/**
 * "2 active" on the Approval workflows card of the Settings page (002 FR-001); nothing until the
 * workflows have arrived.
 */
export function ActiveWorkflowsCount() {
  const workflows = useWorkflows();
  if (!workflows.data) {
    return null;
  }
  return `${workflows.data.items.filter((workflow) => workflow.active).length} active`;
}
