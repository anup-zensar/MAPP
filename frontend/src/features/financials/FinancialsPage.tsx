import { History, Lock, ReceiptText, SearchX, Upload } from 'lucide-react';
import { useEffect, useEffectEvent, useId, useState } from 'react';
import { Outlet, useLocation, useSearchParams } from 'react-router';
import { ApiError } from '@/api/errors';
import { Button } from '@/ui/Button/Button';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { FilterChips } from '@/ui/FilterChips/FilterChips';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { SearchField } from '@/ui/SearchField/SearchField';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { SummaryCard } from '@/ui/SummaryCard/SummaryCard';
import { Tabs } from '@/ui/Tabs/Tabs';
import type { FilterChip } from '@/ui/FilterChips/FilterChips';
import type { TabItem } from '@/ui/Tabs/Tabs';
import { useAccess } from '@/session/access';
import { usePageTitle } from '@/ui/usePageTitle';
import { useFormatters } from '@/format/useFormatters';
import { useFinancialTransactions } from './api';
import type { FinancialRecordRef, FinancialTransactionPage } from './api';
import { DEFAULT_FILTERS, readFilters, TABS, writeFilters } from './financials-links';
import type { FinancialFilters, FinancialTab } from './financials-links';
import {
  accruedLabel,
  countLine,
  NO_ACCESS,
  NO_DATA,
  NO_MATCH,
  PAGE_SUBTITLE,
  SEARCH_PLACEHOLDER,
  sourceLine,
  TAB_LABELS,
  tabLabel,
  TILE_TIPS,
} from './financials-text';
import { ImportDialog } from './ImportDialog';
import { ImportHistoryPanel } from './ImportHistoryPanel';
import { Pager } from './Pager';
import { TransactionsTable } from './TransactionsTable';
import { useNarrow } from './useNarrow';
import styles from './FinancialsPage.module.css';

/** How long the search waits after the last key before it filters. */
export const SEARCH_DELAY_MS = 300;

const LEVEL_WORDS: Record<FinancialRecordRef['type'], string> = {
  plan: 'Plan',
  campaign: 'Campaign',
  initiative: 'Initiative',
  activity: 'Activity',
  lineItem: 'Line item',
};

/**
 * Financials (015 US3, FR-020 – FR-022), at /financials: every purchase order, invoice, payment and
 * accrual GSAP has sent, with tabs by type, search, year, plan and the record filter of the cards'
 * links, the four totals for the filters, and the table, paged by 50, 100 or 200. Everything is
 * kept in the address. A row opens its details panel at /financials/transactions/<id> (the child
 * route renders in the outlet). Import GSAP extract and Import history are in the header (US4).
 */
export function FinancialsPage() {
  usePageTitle('Financials');
  const [params, setParams] = useSearchParams();
  const location = useLocation();
  const filters = readFilters(params);
  const transactions = useFinancialTransactions(filters);
  const narrow = useNarrow();
  const [open, setOpen] = useState<'import' | 'history' | null>(null);
  // 014 FR-011: Import GSAP extract only for a person who may import (finance.import).
  const canImport = useAccess().has('finance.import');
  const data = transactions.data;
  const forbidden =
    transactions.error instanceof ApiError && transactions.error.kind === 'forbidden';

  /** New filters: back to page 1 unless the page itself changes; typing replaces the address. */
  function change(patch: Partial<FinancialFilters>, replace = false) {
    setParams(writeFilters({ ...filters, page: 1, ...patch }), { replace });
  }

  return (
    <>
      <PageHeader
        title="Financials"
        description={PAGE_SUBTITLE}
        actions={
          <>
            <Button variant="tertiary" icon={<History />} onClick={() => setOpen('history')}>
              Import history
            </Button>
            {canImport && (
              <Button icon={<Upload />} onClick={() => setOpen('import')}>
                Import GSAP extract
              </Button>
            )}
          </>
        }
      />
      {forbidden ? (
        <div className={styles.panel}>
          <EmptyState icon={<Lock />} title={NO_ACCESS} />
        </div>
      ) : (
        <div className={styles.sections}>
          <SourceLine page={data} />
          <Totals page={data} failed={transactions.isError && !data} />
          <Tabs
            label="Transaction type"
            tabs={TABS.map((tab): TabItem<FinancialTab> => ({
              value: tab,
              label: tabLabel(tab, data?.counts),
            }))}
            value={filters.tab}
            onChange={(tab) => change({ tab })}
          >
            <section className={styles.panel} aria-label="Transactions">
              <Toolbar filters={filters} page={data} onChange={change} />
              {data ? (
                data.items.length === 0 ? (
                  filtersInUse(filters) ? (
                    <EmptyState
                      icon={<SearchX />}
                      title={NO_MATCH}
                      action={
                        <Button onClick={() => change(clearedFilters(filters))}>
                          Clear filters
                        </Button>
                      }
                    />
                  ) : (
                    <EmptyState icon={<ReceiptText />} title={emptyTabText(filters.tab, data)} />
                  )
                ) : (
                  <>
                    <TransactionsTable
                      rows={data.items}
                      sort={filters.sort}
                      direction={filters.direction}
                      onSort={(sort, direction) => change({ sort, direction })}
                      search={location.search}
                      narrow={narrow}
                    />
                    <Pager
                      page={data.page}
                      pageSize={filters.pageSize}
                      total={data.total}
                      onPage={(page) => change({ page })}
                      onPageSize={(pageSize) => change({ pageSize })}
                    />
                  </>
                )
              ) : transactions.isError ? (
                <div className={styles.body}>
                  <ErrorState
                    error={transactions.error}
                    onRetry={() => void transactions.refetch()}
                  />
                </div>
              ) : (
                <div className={styles.body}>
                  <Skeleton lines={6} height="20px" />
                </div>
              )}
            </section>
          </Tabs>
        </div>
      )}

      <Outlet />
      {open === 'import' && <ImportDialog onClose={() => setOpen(null)} />}
      {open === 'history' && <ImportHistoryPanel onClose={() => setOpen(null)} />}
    </>
  );
}

/** "GSAP data as of 19 Oct 2026, 02:00 · Demo data (simulated GSAP)" under the header (FR-008). */
function SourceLine({ page }: { page: FinancialTransactionPage | undefined }) {
  const { dateTime } = useFormatters();
  if (!page) {
    return null;
  }
  return <p className={styles.source}>{sourceLine(page.source, dateTime)}</p>;
}

/** Committed, Invoiced, Paid and Accrued for the filters, whatever the tab (FR-021). */
function Totals({ page, failed }: { page: FinancialTransactionPage | undefined; failed: boolean }) {
  const { money, moneyCompact } = useFormatters();
  if (failed) {
    return null;
  }
  if (!page) {
    return (
      <div className={styles.cards} aria-hidden="true">
        {[0, 1, 2, 3].map((index) => (
          <div key={index} className={styles.cardSkeleton}>
            <Skeleton lines={2} height="20px" />
          </div>
        ))}
      </div>
    );
  }
  const { totals, source } = page;
  const card = (label: string, value: (typeof totals)['committed'], tip: string) => (
    <SummaryCard
      label={label}
      value={moneyCompact(value)}
      fullValue={money(value)}
      tip={tip}
      tone={value.amount.startsWith('-') ? 'problem' : undefined}
    />
  );

  return (
    <div className={styles.cards}>
      {card('Committed', totals.committed, TILE_TIPS.committed)}
      {card('Invoiced', totals.invoiced, TILE_TIPS.invoiced)}
      {card('Paid', totals.paid, TILE_TIPS.paid)}
      {card(
        accruedLabel(source.accrualMonth),
        totals.accrued,
        TILE_TIPS.accrued(source.accrualMonth),
      )}
    </div>
  );
}

interface ToolbarProps {
  filters: FinancialFilters;
  page: FinancialTransactionPage | undefined;
  onChange: (patch: Partial<FinancialFilters>, replace?: boolean) => void;
}

/** Search, Year and Plan, the count at the end; the record and plan filters as chips below. */
function Toolbar({ filters, page, onChange }: ToolbarProps) {
  const ids = useId();
  const choices = page?.choices;
  const year = filters.year ?? page?.year ?? choices?.currentYear;
  const years = withChosen(choices?.years ?? [], year);
  const plans = choices?.plans ?? [];
  const planName = plans.find((plan) => plan.id === filters.planId)?.name ?? filters.planId;

  const chips: FilterChip[] = [];
  if (filters.recordId) {
    const record = page?.recordFilter;
    chips.push({
      key: 'recordId',
      label: record
        ? `${LEVEL_WORDS[record.type]}: ${record.name} (${record.id})`
        : `Record: ${filters.recordId}`,
    });
  }
  if (filters.planId) {
    chips.push({ key: 'planId', label: `Plan: ${planName}` });
  }

  return (
    <div className={styles.toolbar}>
      <div className={styles.row}>
        <div className={styles.search}>
          <DelayedSearch value={filters.q} onChange={(q) => onChange({ q }, true)} />
        </div>
        <div className={styles.field}>
          <label htmlFor={`${ids}-year`}>Year</label>
          <select
            id={`${ids}-year`}
            className={`${styles.control} ${styles.select} ${styles.year}`}
            value={year ?? ''}
            onChange={(event) => onChange({ year: Number(event.target.value), planId: '' })}
          >
            {years.map((choice) => (
              <option key={choice} value={choice}>
                {choice}
              </option>
            ))}
          </select>
        </div>
        <div className={styles.field}>
          <label htmlFor={`${ids}-plan`}>Plan</label>
          <select
            id={`${ids}-plan`}
            className={`${styles.control} ${styles.select}`}
            value={filters.planId}
            onChange={(event) => onChange({ planId: event.target.value })}
          >
            <option value="">All plans</option>
            {plans.map((plan) => (
              <option key={plan.id} value={plan.id}>
                {plan.name}
              </option>
            ))}
            {filters.planId && !plans.some((plan) => plan.id === filters.planId) && (
              <option value={filters.planId}>{filters.planId}</option>
            )}
          </select>
        </div>
        {page && (
          <p className={styles.count} aria-live="polite">
            {countLine(page.total, page.counts.all)}
          </p>
        )}
      </div>
      <FilterChips
        chips={chips}
        onRemove={(key) => onChange({ [key]: '' })}
        onClearAll={() => onChange(clearedFilters(filters))}
      />
    </div>
  );
}

/** The years to offer, with the chosen one even when no plan or transaction has it. */
function withChosen(years: number[], chosen: number | undefined): number[] {
  return chosen === undefined || years.includes(chosen)
    ? years
    : [...years, chosen].sort((a, b) => b - a);
}

/** What an empty tab says when no filter is in use. */
function emptyTabText(tab: FinancialTab, page: FinancialTransactionPage): string {
  if (!page.source.dataAsOf) {
    return NO_DATA;
  }
  if (tab === 'notLinked') {
    return `Every transaction of ${page.year} is linked to a line item.`;
  }
  const what = tab === 'all' ? 'transactions' : TAB_LABELS[tab].toLowerCase();
  return `No ${what} for ${page.year} yet.`;
}

/** A search, year, plan or record filter is in use (the tab and sort are not filters). */
function filtersInUse(filters: FinancialFilters): boolean {
  return (
    filters.q.trim() !== '' ||
    filters.year !== null ||
    filters.planId !== '' ||
    filters.recordId !== ''
  );
}

/** Clear filters: the search, year, plan and record filter go; the tab and sort stay. */
function clearedFilters(filters: FinancialFilters): Partial<FinancialFilters> {
  return {
    ...DEFAULT_FILTERS,
    tab: filters.tab,
    sort: filters.sort,
    direction: filters.direction,
    pageSize: filters.pageSize,
  };
}

/**
 * The search box: it filters 300 ms after the last key, and shows the address's search again when
 * that changes elsewhere (Clear filters, Back).
 */
function DelayedSearch({ value, onChange }: { value: string; onChange: (q: string) => void }) {
  const [draft, setDraft] = useState<{ text: string; over: string } | null>(null);
  const text = draft && draft.over === value ? draft.text : value;
  const apply = useEffectEvent((q: string) => onChange(q));

  useEffect(() => {
    if (!draft || draft.over !== value || draft.text === value) {
      return;
    }
    const timer = setTimeout(() => apply(draft.text), SEARCH_DELAY_MS);
    return () => clearTimeout(timer);
  }, [draft, value]);

  return (
    <SearchField
      label="Search transactions"
      placeholder={SEARCH_PLACEHOLDER}
      value={text}
      onChange={(next) => setDraft({ text: next, over: value })}
    />
  );
}
