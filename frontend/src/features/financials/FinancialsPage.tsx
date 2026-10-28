import { History, Lock, ReceiptText, SearchX, Upload } from 'lucide-react';
import { useState } from 'react';
import { Outlet, useLocation, useSearchParams } from 'react-router';
import { ApiError } from '@/api/errors';
import { Button } from '@/ui/Button/Button';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { Tabs } from '@/ui/Tabs/Tabs';
import type { TabItem } from '@/ui/Tabs/Tabs';
import { useAccess } from '@/session/access';
import { usePageTitle } from '@/ui/usePageTitle';
import { useFormatters } from '@/format/useFormatters';
import { SimulatedGsapPanel } from '@/features/simulated-gsap';
import { useFinancialTransactions } from './api';
import type { FinancialTransactionPage } from './api';
import { clearedFilters, filtersInUse, TABS, writeFilters } from './financials-links';
import type { FinancialFilters, FinancialTab } from './financials-links';
import {
  NO_ACCESS,
  NO_DATA,
  NO_MATCH,
  PAGE_SUBTITLE,
  sourceLine,
  TAB_LABELS,
  tabLabel,
} from './financials-text';
import { FinancialsToolbar } from './FinancialsToolbar';
import { FinancialTotals } from './FinancialTotals';
import { ImportDialog } from './ImportDialog';
import { ImportHistoryPanel } from './ImportHistoryPanel';
import { Pager } from './Pager';
import { readFilters } from './schemas';
import { TransactionsTable } from './TransactionsTable';
import { useNarrow } from './useNarrow';
import styles from './FinancialsPage.module.css';

/**
 * Financials (015 US3, FR-020 – FR-022): GSAP's transactions by type with search, filters, totals and
 * paging, all kept in the address; a row opens its panel in the outlet. Importers also see the
 * simulated GSAP's SWBS runs (030 FR-013).
 */
export function FinancialsPage() {
  usePageTitle('Financials');
  const [params, setParams] = useSearchParams();
  const location = useLocation();
  const filters = readFilters(params);
  const transactions = useFinancialTransactions(filters);
  const narrow = useNarrow();
  const [open, setOpen] = useState<'import' | 'history' | null>(null);
  // 014 FR-011, 030 FR-013: importing and the simulated GSAP only for a person who may import.
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
          <FinancialTotals page={data} failed={transactions.isError && !data} />
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
              <FinancialsToolbar filters={filters} page={data} onChange={change} />
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
          {canImport && <SimulatedGsapPanel />}
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
