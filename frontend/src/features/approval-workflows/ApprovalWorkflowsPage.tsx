import { CircleAlert, Plus, Workflow } from 'lucide-react';
import { useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router';
import { conflictMessage } from '@/api/errors';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { Button } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import { useToast } from '@/ui/Toast/useToast';
import { Tooltip } from '@/ui/Tooltip/Tooltip';
import { asApiError, isSessionLost } from '@/features/approvals';
import { ListToolbar } from '@/ui/ListToolbar/ListToolbar';
import list from '@/ui/settings-table.module.css';
import { filterRecords, useListFilters } from '@/ui/useListFilters';
import type { ListFilters } from '@/ui/useListFilters';
import { useStatusFlow } from '@/ui/useStatusFlow';
import { useFormatters } from '@/format/useFormatters';
import { usePageTitle } from '@/ui/usePageTitle';
import { useUndoableDelete } from '@/ui/useUndoableDelete';
import { workflowMoreItems } from './workflowActions';
import type { WorkflowActions } from './workflowActions';
import { WorkflowDetailsPanel } from './WorkflowDetailsPanel';
import { WorkflowFormPanel } from './WorkflowFormPanel';
import { cantDeleteText, setInactiveEffects, usedByText } from './workflowTexts';
import {
  WORKFLOWS_KEY,
  useDeleteWorkflow,
  useSetWorkflowActive,
  useWorkflows,
} from './workflowsApi';
import type { ApprovalWorkflow } from './workflowsApi';
import styles from './ApprovalWorkflowsPage.module.css';
import { CanChange, SettingsViewOnlyNote } from '@/ui/CanChange/CanChange';
import { useCanChange } from '@/ui/useCanChange';

const WORKFLOW_NOUN = { one: 'approval workflow', many: 'approval workflows' };

const LIST_PATH = '/settings/approval-workflows';

/** The side panel that is open besides the details (which have their own address). */
type OpenForm = { kind: 'new' } | { kind: 'edit'; workflow: ApprovalWorkflow };

/** The question that waits for its answer. */
type PendingDialog =
  | { kind: 'cantDelete'; workflow: ApprovalWorkflow }
  | { kind: 'delete'; workflow: ApprovalWorkflow };

/**
 * Settings › Approval workflows (US4, FR-001 – FR-008): the Settings list toolbar (search, Active ·
 * Inactive · All, "2 of 2 approval workflows"), the list with each workflow's levels, use and
 * status, and its "⋯" actions; details at /settings/approval-workflows/:workflowId; create and edit
 * in a side panel; a workflow in use is set inactive instead of deleted, with the one set-inactive
 * behaviour of Settings (design system §6.7).
 */
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

  // In use (activities use it or wait with it), the question lists the effects first (FR-007).
  const status = useStatusFlow<ApprovalWorkflow>({
    words: { title: 'Approval workflow', noun: 'approval workflow' },
    usage: (workflow) =>
      workflow.usedByCount > 0 || workflow.waitingCount > 0
        ? setInactiveEffects(workflow.usedByCount, workflow.waitingCount)
        : [],
    effect: () => [],
    request: (workflow, active, version) =>
      setActive.mutateAsync({ id: workflow.id, active, version }),
    onFailure: showFailure,
  });

  const actions: WorkflowActions = {
    edit: (workflow) => {
      if (workflowId) {
        void navigate(LIST_PATH);
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
          onClose={() => void navigate(LIST_PATH)}
        />
      )}
      {form?.kind === 'new' && <WorkflowFormPanel onClose={() => setForm(null)} />}
      {form?.kind === 'edit' && (
        <WorkflowFormPanel workflow={form.workflow} onClose={() => setForm(null)} />
      )}

      {dialog?.kind === 'cantDelete' && (
        <ConfirmDialog
          open
          onOpenChange={(open) => !open && setDialog(null)}
          title={`Can't delete '${dialog.workflow.name}'`}
          confirmLabel={dialog.workflow.active ? 'Set inactive' : 'Close'}
          busy={status.busy}
          onConfirm={() =>
            dialog.workflow.active
              ? void status.applyInactive(dialog.workflow).then(() => setDialog(null))
              : setDialog(null)
          }
        >
          {cantDeleteText(dialog.workflow.usedByCount)}
        </ConfirmDialog>
      )}
      {dialog?.kind === 'delete' && (
        <DeleteDialog
          workflow={dialog.workflow}
          onInUse={(workflow) => setDialog({ kind: 'cantDelete', workflow })}
          onDeleted={() => {
            if (workflowId) {
              void navigate(LIST_PATH);
            }
          }}
          onClose={() => setDialog(null)}
        />
      )}
      {status.dialog}
    </>
  );
}

interface WorkflowListProps {
  all: ApprovalWorkflow[];
  filters: ListFilters;
  onNew: () => void;
  actions: WorkflowActions;
}

/** The toolbar and the table, or what to do when there is nothing to show. */
function WorkflowList({ all, filters, onNew, actions }: WorkflowListProps) {
  if (all.length === 0) {
    return (
      <EmptyState
        icon={<Workflow />}
        title="No approval workflows yet — create your first approval workflow"
        action={
          <CanChange permission="workflows.manage">
            <Button icon={<Plus />} onClick={onNew}>
              New approval workflow
            </Button>
          </CanChange>
        }
      />
    );
  }
  const { shown, counts } = filterRecords(all, filters, (workflow) => workflow.name);

  return (
    <>
      <ListToolbar
        search={{
          label: 'Search approval workflows',
          placeholder: 'Search name',
          value: filters.q,
          onChange: filters.setQ,
        }}
        status={{ value: filters.status, onChange: filters.setStatus, counts }}
        count={{ shown: shown.length, total: all.length, noun: WORKFLOW_NOUN }}
      />
      {shown.length === 0 ? (
        <div className={list.noMatch}>
          <p>Nothing matches these filters</p>
          <Button onClick={filters.clear}>Clear filters</Button>
        </div>
      ) : (
        <div className={list.tableScroll}>
          <table className={list.table}>
            <thead>
              <tr>
                <th scope="col" className={styles.nameCol}>
                  Name
                </th>
                <th scope="col" className={styles.levelsCol}>
                  Levels
                </th>
                <th scope="col" className={styles.usedByCol}>
                  Used by
                </th>
                <th scope="col">Status</th>
                <th scope="col" className={styles.actionsCell}>
                  <span className="visually-hidden">Actions</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {shown.map((workflow) => (
                <WorkflowRow key={workflow.id} workflow={workflow} actions={actions} />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </>
  );
}

function WorkflowRow({
  workflow,
  actions,
}: {
  workflow: ApprovalWorkflow;
  actions: WorkflowActions;
}) {
  const canChange = useCanChange('workflows.manage');
  const navigate = useNavigate();
  const details = `${LIST_PATH}/${workflow.id}`;

  return (
    <tr>
      <th scope="row" className={list.nameCell}>
        <span className={styles.name}>
          <Link to={details}>{workflow.name}</Link>
          {workflow.warnings.length > 0 && (
            <Tooltip content={workflow.warnings.join(' · ')}>
              <button
                type="button"
                className={styles.warning}
                aria-label={`Warnings: ${workflow.warnings.join(', ')}`}
              >
                <CircleAlert aria-hidden="true" />
              </button>
            </Tooltip>
          )}
        </span>
      </th>
      <td>{workflow.levelsText}</td>
      <td>{usedByText(workflow.usedByCount, workflow.waitingCount)}</td>
      <td className={list.nowrap}>
        {workflow.active ? (
          <StatusChip tone="positive">Active</StatusChip>
        ) : (
          <StatusChip tone="neutral">Inactive</StatusChip>
        )}
      </td>
      <td className={list.actionsCell}>
        <ActionsMenu
          label={workflow.name}
          items={[
            { label: 'View', onSelect: () => void navigate(details) },
            ...(canChange
              ? [
                  { label: 'Edit', onSelect: () => actions.edit(workflow) },
                  ...workflowMoreItems(workflow, actions),
                ]
              : []),
          ]}
        />
      </td>
    </tr>
  );
}

interface DeleteDialogProps {
  workflow: ApprovalWorkflow;
  /** The API says the workflow is in use after all. */
  onInUse: (workflow: ApprovalWorkflow) => void;
  onDeleted: () => void;
  onClose: () => void;
}

/** "Delete approval workflow '<name>'?" — then the message with Undo (FR-008). */
function DeleteDialog({ workflow, onInUse, onDeleted, onClose }: DeleteDialogProps) {
  const { show } = useToast();
  const deleteWorkflow = useDeleteWorkflow();
  const afterDelete = useUndoableDelete({ invalidate: [WORKFLOWS_KEY] });

  async function confirm() {
    if (deleteWorkflow.isPending) {
      return;
    }
    try {
      afterDelete(await deleteWorkflow.mutateAsync(workflow.id));
      onDeleted();
      onClose();
    } catch (error) {
      const failure = asApiError(error);
      if (failure.kind === 'inUse') {
        onInUse(workflow);
        return;
      }
      if (!isSessionLost(failure)) {
        show({ message: failure.message });
      }
      onClose();
    }
  }

  return (
    <ConfirmDialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={`Delete approval workflow '${workflow.name}'?`}
      confirmLabel="Delete approval workflow"
      tone="danger"
      busy={deleteWorkflow.isPending}
      onConfirm={() => void confirm()}
    >
      It moves to the recycle bin; you can restore it.
    </ConfirmDialog>
  );
}
