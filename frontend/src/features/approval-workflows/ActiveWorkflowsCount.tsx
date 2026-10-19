import { SettingsCount } from '@/features/settings/shared/SettingsCount';
import { useWorkflows } from './workflowsApi';

/**
 * "2 active" on the Approval workflows card of the Settings page (002 FR-001), as every Settings
 * card shows its count.
 */
export function ActiveWorkflowsCount() {
  const workflows = useWorkflows();
  return (
    <SettingsCount
      loading={workflows.isPending}
      count={workflows.data?.items.filter((workflow) => workflow.active).length}
    />
  );
}
