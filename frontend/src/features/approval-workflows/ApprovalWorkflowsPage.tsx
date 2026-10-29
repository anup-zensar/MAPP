import { Plus } from 'lucide-react';
import { useState } from 'react';
import { useNavigate, useParams } from 'react-router';
import { conflictMessage } from '@/api/errors';
import { Button } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { useToast } from '@/ui/Toast/useToast';
import { asApiError, isSessionLost } from '@/api/errors';
import { useListFilters } from '@/ui/useListFilters';
import { useStatusFlow } from '@/ui/useStatusFlow';
import { useFormatters } from '@/format/useFormatters';
import { usePageTitle } from '@/ui/usePageTitle';
import { CanChange, SettingsViewOnlyNote } from '@/ui/CanChange/CanChange';
import { DeleteWorkflowDialog } from './DeleteWorkflowDialog';
import { WORKFLOWS_PATH } from './workflow-actions';
import type { WorkflowActions } from './workflow-actions';
import { WorkflowDetailsPanel } from './WorkflowDetailsPanel';
import { WorkflowFormPanel } from './WorkflowFormPanel';
import { WorkflowList } from './WorkflowList';
import { cantDeleteText, setInactiveEffects } from './workflow-texts';
import { useSetWorkflowActive, useWorkflows } from './api';
import type { ApprovalWorkflow } from './api';

/** The side panel that is open besides the details (which have their own address). */
type OpenForm = { kind: 'new' } | { kind: 'edit'; workflow: ApprovalWorkflow };

type PendingDialog =
  | { kind: 'cantDelete'; workflow: ApprovalWorkflow }
  | { kind: 'delete'; workflow: ApprovalWorkflow };

/** A workflow in use is set inactive instead of deleted (design system §6.7). */
export function ApprovalWorkflowsPage() {
  usePageTitle('Approval workflows');
  const { workflowId } = useParams();
  const navigate = useNavigate();
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const workflows = useWorkflows();
  const setActive = useSetWorkflowActive();
  const filters = useListFilters();
  const [form, setForm] = useState<OpenForm | null>(null);
  const [dialog, setDialog] = useState<PendingDialog | null>(null);

  function showFailure(error: unknown) {
    const failure = asApiError(error);
    if (isSessionLost(failure)) {
      return;
    }
    show({
      message:
        failure.kind === 'conflict'
          ? conflictMessage(failure, 'approval workflow', dateFormat)
          : failure.message,
    });
  }

  // In use, the question lists the effects first.
  const status = useStatusFlow<ApprovalWorkflow>({
    words: { title: 'Approval workflow', noun: 'approval workflow' },
    usage: (workflow) =>
      workflow.usedByCount > 0 || workflow.waitingCount > 0 ? setInactiveEffects(workflow) : [],
    effect: () => [],
    request: (workflow, active, version) =>
      setActive.mutateAsync({ id: workflow.id, active, version }),
    onFailure: showFailure,
  });

  const actions: WorkflowActions = {
    edit: (workflow) => {
      if (workflowId) {
        void navigate(WORKFLOWS_PATH);
      }
      setForm({ kind: 'edit', workflow });
    },
    setInactive: (workflow) => status.setInactive(workflow),
    setActive: (workflow) => status.reactivate(workflow),
    remove: (workflow) =>
      setDialog({ kind: workflow.usedByCount > 0 ? 'cantDelete' : 'delete', workflow }),
  };

  return (
    <>
      <PageHeader
        breadcrumb={[{ label: 'Settings', to: '/settings' }, { label: 'Approval workflows' }]}
        title="Approval workflows"
        description="Who approves activities, level by level."
        actions={
          <CanChange permission="workflows.manage">
            <Button variant="primary" icon={<Plus />} onClick={() => setForm({ kind: 'new' })}>
              New approval workflow
            </Button>
          </CanChange>
        }
      />
      <SettingsViewOnlyNote permission="workflows.manage" />
      <Card>
        {workflows.data ? (
          <WorkflowList
            all={workflows.data.items}
            filters={filters}
            onNew={() => setForm({ kind: 'new' })}
            actions={actions}
          />
        ) : workflows.isError ? (
          <ErrorState error={workflows.error} onRetry={() => void workflows.refetch()} />
        ) : (
          <Skeleton lines={5} height="18px" />
        )}
      </Card>

      {workflowId && !form && (
        <WorkflowDetailsPanel
          workflowId={workflowId}
          actions={actions}
          onClose={() => void navigate(WORKFLOWS_PATH)}
        />
      )}
      {form?.kind === 'new' && <WorkflowFormPanel onClose={() => setForm(null)} />}
      {form?.kind === 'edit' && (
        <WorkflowFormPanel workflow={form.workflow} onClose={() => setForm(null)} />
      )}

      {dialog?.kind === 'cantDelete' && dialog.workflow.active && (
        <ConfirmDialog
          open
          onOpenChange={(open) => !open && setDialog(null)}
          title={`Can't delete '${dialog.workflow.name}'`}
          confirmLabel="Set inactive"
          busy={status.busy}
          onConfirm={() => void status.applyInactive(dialog.workflow).then(() => setDialog(null))}
        >
          {cantDeleteText(dialog.workflow)}
        </ConfirmDialog>
      )}
      {/* Already inactive: the dialog only informs, with one Close (038 FR-044). */}
      {dialog?.kind === 'cantDelete' && !dialog.workflow.active && (
        <ConfirmDialog
          open
          onOpenChange={(open) => !open && setDialog(null)}
          title={`Can't delete '${dialog.workflow.name}'`}
          acknowledgeOnly
        >
          {cantDeleteText(dialog.workflow)}
        </ConfirmDialog>
      )}
      {dialog?.kind === 'delete' && (
        <DeleteWorkflowDialog
          workflow={dialog.workflow}
          onInUse={(workflow) => setDialog({ kind: 'cantDelete', workflow })}
          onDeleted={() => {
            if (workflowId) {
              void navigate(WORKFLOWS_PATH);
            }
          }}
          onClose={() => setDialog(null)}
        />
      )}
      {status.dialog}
    </>
  );
}
