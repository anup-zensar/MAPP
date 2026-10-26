import { Pencil, Plus, Wallet } from 'lucide-react';
import { useState } from 'react';
import { useSearchParams } from 'react-router';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { Button } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { SegmentedFilter } from '@/ui/SegmentedFilter/SegmentedFilter';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import { plural } from '@/lib/text';
import { usePageTitle } from '@/lib/usePageTitle';
import { ListToolbar } from '../shared/ListToolbar';
import { SettingDeleteDialog } from '../shared/SettingDeleteDialog';
import styles from '../shared/SettingsTable.module.css';
import { filterRecords, useListFilters } from '../shared/useListFilters';
import type { ListFilters } from '../shared/useListFilters';
import { useSettingStatus } from '../shared/useSettingStatus';
import { FinanceBudgetItemDetails } from './FinanceBudgetItemDetails';
import { FinanceBudgetItemForm } from './FinanceBudgetItemForm';
import page from './FinanceBudgetItemsPage.module.css';
import {
  FINANCE_LISTS,
  setFinanceBudgetItemStatus,
  useDeleteFinanceBudgetItem,
  useFinanceBudgetPage,
} from './financeBudgetsApi';
import type { FinanceBudget, FinanceBudgetItem, FinanceBudgetPage } from './financeBudgetsApi';
import { moreActions } from './financeItemActions';
import type { FinanceItemActions } from './financeItemActions';
import { RenameBudgetDialog } from './RenameBudgetDialog';
import { CanChange, SettingsViewOnlyNote } from '../shared/CanChange';
import { useCanChange } from '../shared/useCanChange';

const ITEM_NOUN = { one: 'finance budget item', many: 'finance budget items' };

/** The side-panel form that is open: a new item, or one being changed. */
type OpenForm = { kind: 'new' } | { kind: 'edit'; item: FinanceBudgetItem };

/** What the search looks in: the item, its budget and its code (FR-014). */
function searchText(item: FinanceBudgetItem): string {
  return `${item.name} ${item.budgetName} ${item.code}`;
}

/**
 * Finance budget items (feature 002, US5, FR-014, FR-015): the items line items link to, a simple
 * list until the Finance module arrives. A budget year filter (the current year first), search and
 * status in the address, one card per finance budget with its items, Rename budget, details in a
 * side panel (`?id=`), New and Edit in a side-panel form, Set inactive / Reactivate and Delete
 * (unused items only, with Undo).
 */
export function FinanceBudgetItemsPage() {
  usePageTitle('Finance budget items');
  const filters = useListFilters();
  const [params, setParams] = useSearchParams();
  const yearParam = Number.parseInt(params.get('year') ?? '', 10);
  const requestedYear = Number.isInteger(yearParam) ? yearParam : undefined;
  const budgets = useFinanceBudgetPage(requestedYear);
  const deleteItem = useDeleteFinanceBudgetItem();
  const [form, setForm] = useState<OpenForm | null>(null);
  const [renaming, setRenaming] = useState<FinanceBudget | null>(null);
  const [deleting, setDeleting] = useState<FinanceBudgetItem | null>(null);
  const status = useSettingStatus({
    kind: 'financeBudgetItem',
    request: setFinanceBudgetItemStatus,
    invalidate: FINANCE_LISTS,
  });

  const actions: FinanceItemActions = {
    view: (item) => filters.setOpenId(item.id),
    edit: (item) => setForm({ kind: 'edit', item }),
    setActive: (item, active) => (active ? status.reactivate(item) : status.setInactive(item)),
    remove: (item) => setDeleting(item),
  };
  const newItem = () => setForm({ kind: 'new' });
  const data = budgets.data;
  // The year chosen, also while its budgets are still on their way.
  const year = requestedYear ?? data?.year;

  function chooseYear(next: string) {
    setParams(
      (previous) => {
        const changed = new URLSearchParams(previous);
        changed.set('year', next);
        changed.delete('id');
        return changed;
      },
      { replace: true },
    );
  }

  // The details wait for the list; an item that is not in it any more says so.
  const openItem =
    filters.openId && data
      ? (data.items.flatMap((budget) => budget.items).find((item) => item.id === filters.openId) ??
        null)
      : undefined;
  const openBudget = openItem
    ? data?.items.find((budget) => budget.id === openItem.budgetId)
    : undefined;

  return (
    <>
      <PageHeader
        breadcrumb={[{ label: 'Settings', to: '/settings' }, { label: 'Finance budget items' }]}
        title="Finance budget items"
        description="The finance budget items line items link to. A simple list until the Finance module arrives."
        actions={
          <CanChange>
            <Button variant="primary" icon={<Plus />} onClick={newItem}>
              New finance budget item
            </Button>
          </CanChange>
        }
      />
      <SettingsViewOnlyNote />
      {!data ? (
        <Card>
          {budgets.isError ? (
            <ErrorState error={budgets.error} onRetry={() => void budgets.refetch()} />
          ) : (
            <Skeleton lines={5} height="18px" />
          )}
        </Card>
      ) : (
        <BudgetCards
          data={data}
          year={year ?? data.year}
          loading={budgets.isPlaceholderData}
          filters={filters}
          actions={actions}
          onYear={chooseYear}
          onNew={newItem}
          onRename={setRenaming}
        />
      )}

      {form && data && (
        <FinanceBudgetItemForm
          item={form.kind === 'edit' ? form.item : undefined}
          year={year ?? data.year}
          years={data.years}
          onClose={() => setForm(null)}
        />
      )}
      {!form && openItem !== undefined && (
        <FinanceBudgetItemDetails
          item={openItem}
          budget={openBudget}
          actions={actions}
          onClose={() => filters.setOpenId('')}
        />
      )}
      {renaming && <RenameBudgetDialog budget={renaming} onClose={() => setRenaming(null)} />}
      {deleting && (
        <SettingDeleteDialog
          kind="financeBudgetItem"
          name={deleting.name}
          usage={deleting.usage}
          active={deleting.active}
          remove={() => deleteItem.mutateAsync(deleting.id)}
          setInactive={() => status.applyInactive(deleting)}
          invalidate={FINANCE_LISTS}
          onDeleted={() => {
            if (filters.openId === deleting.id) {
              filters.setOpenId('');
            }
          }}
          onClose={() => setDeleting(null)}
        />
      )}
      {status.dialog}
    </>
  );
}

interface BudgetCardsProps {
  data: FinanceBudgetPage;
  /** The year chosen in the filter. */
  year: number;
  /** Another year's budgets are on their way; the last ones are shown meanwhile. */
  loading: boolean;
  filters: ListFilters;
  actions: FinanceItemActions;
  onYear: (year: string) => void;
  onNew: () => void;
  onRename: (budget: FinanceBudget) => void;
}

/** The filters, then one card per finance budget of the year — or what to show instead. */
function BudgetCards({
  data,
  year,
  loading,
  filters,
  actions,
  onYear,
  onNew,
  onRename,
}: BudgetCardsProps) {
  const all = data.items.flatMap((budget) => budget.items);
  const { counts } = filterRecords(all, filters, searchText);
  const shown = data.items
    .map((budget) => ({ budget, items: filterRecords(budget.items, filters, searchText).shown }))
    .filter((group) => group.items.length > 0);
  const shownCount = shown.reduce((sum, group) => sum + group.items.length, 0);

  return (
    <div className={page.stack} aria-busy={loading || undefined}>
      <Card>
        <div className={page.filters}>
          <SegmentedFilter
            label="Budget year"
            options={data.years.map((each) => ({ value: String(each), label: String(each) }))}
            value={String(year)}
            onChange={onYear}
          />
          <ListToolbar
            search={{
              label: 'Search finance budget items',
              placeholder: 'Search item, budget or code',
              value: filters.q,
              onChange: filters.setQ,
            }}
            status={{ value: filters.status, onChange: filters.setStatus, counts }}
            count={{ shown: shownCount, total: all.length, noun: ITEM_NOUN }}
          />
        </div>
      </Card>
      {loading ? (
        <Card>
          <Skeleton lines={5} height="18px" />
        </Card>
      ) : all.length === 0 ? (
        <Card>
          <EmptyState
            icon={<Wallet />}
            title={`No finance budget items for ${data.year} yet — create the first one`}
            action={
              <CanChange>
                <Button icon={<Plus />} onClick={onNew}>
                  New finance budget item
                </Button>
              </CanChange>
            }
          />
        </Card>
      ) : shown.length === 0 ? (
        <Card>
          <div className={styles.noMatch}>
            <p>Nothing matches these filters</p>
            <Button onClick={filters.clear}>Clear filters</Button>
          </div>
        </Card>
      ) : (
        shown.map(({ budget, items }) => (
          <BudgetCard
            key={budget.id}
            budget={budget}
            items={items}
            actions={actions}
            onRename={() => onRename(budget)}
          />
        ))
      )}
    </div>
  );
}

interface BudgetCardProps {
  budget: FinanceBudget;
  /** The budget's items that match the filters. */
  items: FinanceBudgetItem[];
  actions: FinanceItemActions;
  onRename: () => void;
}

/** One finance budget: its name, item count and the plans using it, with Rename budget. */
function BudgetCard({ budget, items, actions, onRename }: BudgetCardProps) {
  const canChange = useCanChange();
  return (
    <Card
      title={budget.name}
      subtitle={`${plural(budget.items.length, 'item')} · ${budget.usage.summary}`}
      actions={
        canChange ? (
          <Button
            size="sm"
            icon={<Pencil />}
            aria-label={`Rename budget '${budget.name}'`}
            onClick={onRename}
          >
            Rename budget
          </Button>
        ) : undefined
      }
    >
      <div className={styles.tableScroll}>
        {/* Fixed column widths, so the columns of every budget's table line up (UI audit ST-04). */}
        <table className={`${styles.table} ${styles.fixed}`}>
          <colgroup>
            <col className={page.itemColumn} />
            <col className={page.codeColumn} />
            <col />
            <col className={page.statusColumn} />
            <col className={page.actionsColumn} />
          </colgroup>
          <thead>
            <tr>
              <th scope="col">Item</th>
              <th scope="col">Code</th>
              <th scope="col">Linked to</th>
              <th scope="col">Status</th>
              <th scope="col">
                <span className="visually-hidden">Actions</span>
              </th>
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={item.id}>
                <th scope="row">
                  <button
                    type="button"
                    className={styles.nameButton}
                    onClick={() => actions.view(item)}
                  >
                    {item.name}
                  </button>
                </th>
                <td className={styles.nowrap}>{item.code || '—'}</td>
                <td>{item.usage.summary}</td>
                <td className={styles.nowrap}>
                  <StatusChip tone={item.active ? 'positive' : 'neutral'}>
                    {item.active ? 'Active' : 'Inactive'}
                  </StatusChip>
                </td>
                <td className={styles.actionsCell}>
                  <ActionsMenu
                    label={item.name}
                    items={[
                      { label: 'View', onSelect: () => actions.view(item) },
                      ...(canChange
                        ? [
                            { label: 'Edit', onSelect: () => actions.edit(item) },
                            ...moreActions(item, actions),
                          ]
                        : []),
                    ]}
                  />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Card>
  );
}
