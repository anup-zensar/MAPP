import { CircleAlert, Plus, Workflow } from 'lucide-react';
import { Link, useNavigate } from 'react-router';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { Button } from '@/ui/Button/Button';
import { CanChange } from '@/ui/CanChange/CanChange';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { ListToolbar } from '@/ui/ListToolbar/ListToolbar';
import list from '@/ui/settings-table.module.css';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import { Tooltip } from '@/ui/Tooltip/Tooltip';
import { useCanChange } from '@/ui/useCanChange';
import { filterRecords } from '@/ui/useListFilters';
import type { ListFilters } from '@/ui/useListFilters';
import { WORKFLOWS_PATH, workflowMoreItems } from './workflow-actions';
import type { WorkflowActions } from './workflow-actions';
import { usedByText } from './workflow-texts';
import type { ApprovalWorkflow } from './api';
import styles from './WorkflowList.module.css';

const WORKFLOW_NOUN = { one: 'approval workflow', many: 'approval workflows' };

interface WorkflowListProps {
  all: ApprovalWorkflow[];
  filters: ListFilters;
  onNew: () => void;
  actions: WorkflowActions;
}

/** The toolbar and the table, or what to do when there is nothing to show. */
export function WorkflowList({ all, filters, onNew, actions }: WorkflowListProps) {
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
  const details = `${WORKFLOWS_PATH}/${workflow.id}`;

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
