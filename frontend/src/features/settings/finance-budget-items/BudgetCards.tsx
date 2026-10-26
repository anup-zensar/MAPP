import { Plus, Wallet } from 'lucide-react';
import { Button } from '@/ui/Button/Button';
import { CanChange } from '@/ui/CanChange/CanChange';
import { Card } from '@/ui/Card/Card';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { ListToolbar } from '@/ui/ListToolbar/ListToolbar';
import { SegmentedFilter } from '@/ui/SegmentedFilter/SegmentedFilter';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import styles from '@/ui/settings-table.module.css';
import { filterRecords } from '@/ui/useListFilters';
import type { ListFilters } from '@/ui/useListFilters';
import type { FinanceBudget, FinanceBudgetItem, FinanceBudgetPage } from './api';
import { BudgetCard } from './BudgetCard';
import type { FinanceItemActions } from './finance-item-actions';
import page from './FinanceBudgetItemsPage.module.css';

const ITEM_NOUN = { one: 'finance budget item', many: 'finance budget items' };

export interface BudgetCardsProps {
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
export function BudgetCards({
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

/** What the search looks in: the item, its budget and its code (FR-014). */
function searchText(item: FinanceBudgetItem): string {
  return `${item.name} ${item.budgetName} ${item.code}`;
}
