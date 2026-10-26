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
