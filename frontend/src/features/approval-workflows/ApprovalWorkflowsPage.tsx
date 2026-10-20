import { CircleAlert, Plus, SearchX, Workflow } from 'lucide-react';
import { useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router';
import { conflictMessage } from '@/api/errors';
import {
  ActionsMenu,
  Button,
  Card,
  ConfirmDialog,
  EmptyState,
  ErrorState,
  PageHeader,
  SearchField,
  SegmentedFilter,
  Skeleton,
  StatusChip,
  Tooltip,
  useToast,
} from '@/components';
import { asApiError, isSessionLost } from '@/features/approvals/messages';
import { useFormatters } from '@/lib/useFormatters';
import { usePageTitle } from '@/lib/usePageTitle';
import { useUndoableDelete } from '@/lib/useUndoableDelete';
import { WorkflowDetailsPanel } from './WorkflowDetailsPanel';
import type { WorkflowActions } from './WorkflowDetailsPanel';
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

type StatusFilter = 'active' | 'inactive' | 'all';

const LIST_PATH = '/settings/approval-workflows';

/** The side panel that is open besides the details (which have their own address). */
type OpenForm = { kind: 'new' } | { kind: 'edit'; workflow: ApprovalWorkflow };

/** The question that waits for its answer. */
type PendingDialog =
  | { kind: 'setInactive'; workflow: ApprovalWorkflow }
  | { kind: 'cantDelete'; workflow: ApprovalWorkflow }
  | { kind: 'delete'; workflow: ApprovalWorkflow };

/**
 * Settings › Approval workflows (US4, FR-001 – FR-008): search, All / Active / Inactive, the list
 * with each workflow's levels, use and status, and its "⋯" actions; details at
 * /settings/approval-workflows/:workflowId; create and edit in a side panel; a workflow in use is
 * set inactive instead of deleted.
 */
export function ApprovalWorkflowsPage() {
  usePageTitle('Approval workflows');
  const { workflowId } = useParams();
  const navigate = useNavigate();
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const workflows = useWorkflows();
  const setActive = useSetWorkflowActive();
  const [search, setSearch] = useState('');
  // Settings lists open on the records in use (design system §2.4 D).
  const [status, setStatus] = useState<StatusFilter>('active');
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

  async function changeActive(workflow: ApprovalWorkflow, active: boolean, offerUndo = false) {
    if (setActive.isPending) {
      return;
    }
    try {
      const saved = await setActive.mutateAsync({
        id: workflow.id,
        active,
        version: workflow.version,
      });
      show({
        message: active
          ? `Approval workflow '${workflow.name}' is active again`
          : `Approval workflow '${workflow.name}' set inactive`,
        // Setting an unused workflow inactive applies at once, with Undo (design system §6.7).
        action: offerUndo
          ? {
              label: 'Undo',
              altText: 'Set it active again from its actions menu',
              onClick: () => void changeActive(saved, true),
            }
          : undefined,
      });
    } catch (error) {
      showFailure(error);
    } finally {
      setDialog(null);
    }
  }

  const actions: WorkflowActions = {
    edit: (workflow) => {
      if (workflowId) {
        void navigate(LIST_PATH);
      }
      setForm({ kind: 'edit', workflow });
    },
    // In use: say what changes first; not in use: at once, with Undo (design system §6.7).
    setInactive: (workflow) =>
      workflow.usedByCount > 0
        ? setDialog({ kind: 'setInactive', workflow })
        : void changeActive(workflow, false, true),
    setActive: (workflow) => void changeActive(workflow, true),
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
          <Button variant="primary" icon={<Plus />} onClick={() => setForm({ kind: 'new' })}>
            New approval workflow
          </Button>
        }
      />
      <Card>
        {workflows.data ? (
          <WorkflowList
            all={workflows.data.items}
            search={search}
            status={status}
            onSearch={setSearch}
            onStatus={setStatus}
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

      {dialog?.kind === 'setInactive' && (
        <ConfirmDialog
          open
          onOpenChange={(open) => !open && setDialog(null)}
          title={`Set '${dialog.workflow.name}' inactive?`}
          confirmLabel="Set inactive"
          busy={setActive.isPending}
          onConfirm={() => void changeActive(dialog.workflow, false)}
        >
          {setInactiveEffects(dialog.workflow.usedByCount, dialog.workflow.waitingCount).map(
            (effect) => (
              <p key={effect}>{effect}</p>
            ),
          )}
        </ConfirmDialog>
      )}
      {dialog?.kind === 'cantDelete' && (
        <ConfirmDialog
          open
          onOpenChange={(open) => !open && setDialog(null)}
          title={`Can't delete '${dialog.workflow.name}'`}
          confirmLabel={dialog.workflow.active ? 'Set inactive' : 'Close'}
          busy={setActive.isPending}
          onConfirm={() =>
            dialog.workflow.active ? void changeActive(dialog.workflow, false) : setDialog(null)
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
    </>
  );
}

interface WorkflowListProps {
  all: ApprovalWorkflow[];
  search: string;
  status: StatusFilter;
  onSearch: (search: string) => void;
  onStatus: (status: StatusFilter) => void;
  onNew: () => void;
  actions: WorkflowActions;
}

/** The toolbar and the table, or what to do when there is nothing to show. */
function WorkflowList({
  all,
  search,
  status,
  onSearch,
  onStatus,
  onNew,
  actions,
}: WorkflowListProps) {
  if (all.length === 0) {
    return (
      <EmptyState
        icon={<Workflow />}
        title="No approval workflows yet — create your first approval workflow"
        action={
          <Button icon={<Plus />} onClick={onNew}>
            New approval workflow
          </Button>
        }
      />
    );
  }
  const text = search.trim().toLowerCase();
  const counts = {
    active: all.filter((workflow) => workflow.active).length,
    inactive: all.filter((workflow) => !workflow.active).length,
    all: all.length,
  };
  const shown = all.filter(
    (workflow) =>
      workflow.name.toLowerCase().includes(text) &&
      (status === 'all' || workflow.active === (status === 'active')),
  );

  return (
    <>
      <div className={styles.toolbar}>
        <div className={styles.search}>
          <SearchField
            label="Search name"
            placeholder="Search name"
            value={search}
            onChange={onSearch}
          />
        </div>
        <SegmentedFilter
          label="Status"
          value={status}
          onChange={onStatus}
          options={[
            { value: 'active', label: `Active (${counts.active})` },
            { value: 'inactive', label: `Inactive (${counts.inactive})` },
            { value: 'all', label: `All (${counts.all})` },
          ]}
        />
        <p className={styles.count} aria-live="polite">
          {`${shown.length} of ${workflowsCount(all.length)}`}
        </p>
      </div>
      {shown.length === 0 ? (
        <EmptyState
          icon={<SearchX />}
          title="Nothing matches these filters"
          action={
            <Button
              onClick={() => {
                onSearch('');
                onStatus('all');
              }}
            >
              Clear filters
            </Button>
          }
        />
      ) : (
        <div className={styles.scroll}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th scope="col" className={styles.nameCol}>
                  Name
                </th>
                <th scope="col" className={styles.levelsCol}>
                  Levels
                </th>
                <th scope="col">Used by</th>
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

/** "1 approval workflow" / "3 approval workflows", for the count line "2 of 3 approval workflows". */
function workflowsCount(count: number): string {
  return count === 1 ? '1 approval workflow' : `${count} approval workflows`;
}

function WorkflowRow({
  workflow,
  actions,
}: {
  workflow: ApprovalWorkflow;
  actions: WorkflowActions;
}) {
  const navigate = useNavigate();
  const details = `${LIST_PATH}/${workflow.id}`;

  return (
    <tr>
      <th scope="row">
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
      <td className={styles.nowrap}>
        {workflow.active ? (
          <StatusChip tone="positive">Active</StatusChip>
        ) : (
          <StatusChip tone="neutral">Inactive</StatusChip>
        )}
      </td>
      <td className={styles.actionsCell}>
        <ActionsMenu
          label={workflow.name}
          items={[
            { label: 'View', onSelect: () => void navigate(details) },
            { label: 'Edit', onSelect: () => actions.edit(workflow) },
            workflow.active
              ? { label: 'Set inactive', onSelect: () => actions.setInactive(workflow) }
              : { label: 'Set active', onSelect: () => actions.setActive(workflow) },
            {
              label: 'Delete',
              tone: 'danger',
              separatorBefore: true,
              onSelect: () => actions.remove(workflow),
            },
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
