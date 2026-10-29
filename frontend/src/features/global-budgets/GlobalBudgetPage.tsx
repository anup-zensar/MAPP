import { Landmark, Lock, Pencil } from 'lucide-react';
import { useState } from 'react';
import { useParams } from 'react-router';
import { ApiError } from '@/api/errors';
import { Button, LinkButton } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { usePageTitle } from '@/ui/usePageTitle';
import { AllocationDialog } from './AllocationDialog';
import type { AllocationTarget } from './AllocationDialog';
import { BudgetSummaryCards } from './BudgetFigures';
import { Allocations } from './Allocations';
import { DeleteBudgetDialog } from './DeleteBudgetDialog';
import { GlobalBudgetForm } from './GlobalBudgetForm';
import { RemoveAllocationDialog } from './RemoveAllocationDialog';
import { WithoutAllocation } from './WithoutAllocation';
import { useGlobalBudget } from './api';
import type { BudgetAllocationRow, GlobalBudget } from './api';
import { inUseText } from './global-budgets-text';
import { BUDGET_GONE } from './save-failure';
import styles from './GlobalBudgetPage.module.css';

type Dialog =
  | { kind: 'edit' }
  | { kind: 'allocate'; target: AllocationTarget }
  | { kind: 'remove'; row: BudgetAllocationRow }
  | { kind: 'delete' }
  | { kind: 'cantDelete'; reason: string };

/**
 * A global budget's page (014 US6, FR-032 – FR-035), at /budgets/<id>: its four figures (red
 * "Over-allocated by X", never blocking), one row per allocated plan with teams, owner, status,
 * allocated, planned cost, left to plan and budget used, the "Other markets" row for plans the
 * person can't see, and the plans of its year without an allocation. Allocate, change and remove
 * allocations; edit and delete the global budget.
 */
export function GlobalBudgetPage() {
  const { globalBudgetId = '' } = useParams();
  const query = useGlobalBudget(globalBudgetId);
  const budget = query.data;
  usePageTitle(budget?.name ?? 'Global budget');
  const [dialog, setDialog] = useState<Dialog | null>(null);
  const failure = query.error instanceof ApiError ? query.error : null;

  if (!budget) {
    return (
      <>
        <PageHeader
          breadcrumb={[{ label: 'Global budgets', to: '/budgets' }, { label: 'Global budget' }]}
          title="Global budget"
        />
        <Card>
          {failure?.kind === 'notFound' ? (
            <EmptyState
              icon={<Landmark />}
              title={BUDGET_GONE}
              action={<LinkButton to="/budgets">Go to Global budgets</LinkButton>}
            />
          ) : failure?.kind === 'forbidden' ? (
            <EmptyState icon={<Lock />} title={failure.message} />
          ) : query.isError ? (
            <ErrorState error={query.error} onRetry={() => void query.refetch()} />
          ) : (
            <Skeleton lines={6} height="20px" />
          )}
        </Card>
      </>
    );
  }

  const tryDelete = () =>
    setDialog(
      budget.access.delete.available
        ? { kind: 'delete' }
        : {
            kind: 'cantDelete',
            reason:
              budget.access.delete.reason || inUseText(budget.name, budget.allocations.length),
          },
    );

  return (
    <>
      <PageHeader
        breadcrumb={[{ label: 'Global budgets', to: '/budgets' }, { label: budget.name }]}
        title={budget.name}
        meta={<Facts budget={budget} />}
        description={budget.description || undefined}
        moreActions={[{ label: 'Delete', tone: 'danger', onSelect: tryDelete }]}
        actions={
          <Button variant="primary" icon={<Pencil />} onClick={() => setDialog({ kind: 'edit' })}>
            Edit
          </Button>
        }
      />
      <div className={styles.sections}>
        <BudgetSummaryCards figures={budget.figures} />
        <Allocations
          budget={budget}
          onAllocate={() => setDialog({ kind: 'allocate', target: { kind: 'new' } })}
          onChange={(row) => setDialog({ kind: 'allocate', target: { kind: 'change', row } })}
          onRemove={(row) => setDialog({ kind: 'remove', row })}
        />
        <WithoutAllocation
          budget={budget}
          onAllocate={(option) => setDialog({ kind: 'allocate', target: { kind: 'new', option } })}
        />
      </div>

      {dialog?.kind === 'edit' && (
        <GlobalBudgetForm budget={budget} onClose={() => setDialog(null)} />
      )}
      {dialog?.kind === 'allocate' && (
        <AllocationDialog budget={budget} target={dialog.target} onClose={() => setDialog(null)} />
      )}
      {dialog?.kind === 'remove' && (
        <RemoveAllocationDialog row={dialog.row} onClose={() => setDialog(null)} />
      )}
      {dialog?.kind === 'delete' && (
        <DeleteBudgetDialog
          budget={budget}
          onInUse={(reason) => setDialog({ kind: 'cantDelete', reason })}
          onClose={() => setDialog(null)}
        />
      )}
      {dialog?.kind === 'cantDelete' && (
        <ConfirmDialog
          open
          onOpenChange={(open) => !open && setDialog(null)}
          title={`Can't delete '${budget.name}'`}
          acknowledgeOnly
        >
          {dialog.reason}
        </ConfirmDialog>
      )}
    </>
  );
}

/** The meta row: year, ID and owner. */
function Facts({ budget }: { budget: GlobalBudget }) {
  return (
    <>
      <span>{budget.year}</span>
      <span>{budget.id}</span>
      <span>{budget.owner ? `Owner: ${budget.owner.name}` : 'No owner'}</span>
    </>
  );
}
