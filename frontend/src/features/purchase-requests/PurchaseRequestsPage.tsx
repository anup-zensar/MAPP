import { Ban, ChevronLeft, ChevronRight, FileText, Plus, SearchX } from 'lucide-react';
import { useSearchParams } from 'react-router';
import { ApiError } from '@/api/errors';
import { Button, LinkButton } from '@/ui/Button/Button';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { NoAccess } from '@/ui/NoAccess/NoAccess';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { usePageTitle } from '@/ui/usePageTitle';
import { usePeople } from '@/features/people';
import { useAttributeForm } from '@/features/settings';
import { usePurchaseRequestList } from './api';
import type { PurchaseRequestListPage } from './api';
import type { Choice } from './list-filters';
import {
  CLEAR_FILTERS,
  LIST_DESCRIPTION,
  NO_MATCH,
  NO_PURCHASE_REQUESTS,
  NOTHING_WAITING,
  pageText,
} from './list-text';
import { PurchaseRequestFilters } from './PurchaseRequestFilters';
import { NEW_PURCHASE_REQUEST, PURCHASE_REQUESTS, SWITCHED_OFF } from './purchase-request-text';
import {
  clearedListSearch,
  LIST_PAGE_SIZE,
  listFiltersInUse,
  readListSearch,
  toListQuery,
  writeListSearch,
} from './schemas';
import type { ListSearch } from './schemas';
import { VendorGroupTable } from './VendorGroupTable';
import styles from './PurchaseRequestsPage.module.css';

/**
 * The Purchase requests page (032 US5, FR-044 – FR-048): purchase requests grouped by vendor, with
 * search, filters, sort and pages kept in the address, and the requisitioner's Waiting for GSAP view.
 */
export function PurchaseRequestsPage() {
  usePageTitle(PURCHASE_REQUESTS);
  const [params, setParams] = useSearchParams();
  const search = readListSearch(params);
  const list = usePurchaseRequestList(toListQuery(search));
  const error = list.error instanceof ApiError ? list.error : null;
  const switchedOff = error?.kind === 'notAllowed' || list.data?.purchaseSupport === false;

  /** New filters go back to page 1 unless the page itself changes; typing replaces the address. */
  function change(patch: Partial<ListSearch>, replace = false) {
    setParams(writeListSearch({ ...search, page: 1, ...patch }), { replace });
  }

  if (error?.kind === 'forbidden') {
    return <NoAccess message={error.message} />;
  }

  return (
    <>
      <PageHeader
        title={PURCHASE_REQUESTS}
        description={LIST_DESCRIPTION}
        actions={
          !switchedOff && (
            <LinkButton to="/purchase-requests/new" icon={<Plus />}>
              {NEW_PURCHASE_REQUEST}
            </LinkButton>
          )
        }
      />
      {switchedOff ? (
        <section className={styles.panel} aria-label={PURCHASE_REQUESTS}>
          <EmptyState icon={<Ban />} title={error?.message || SWITCHED_OFF} />
        </section>
      ) : (
        <section className={styles.panel} aria-label={PURCHASE_REQUESTS}>
          <Toolbar search={search} total={list.data?.total} page={list.data} onChange={change} />
          {list.data ? (
            <Results page={list.data} search={search} onChange={change} />
          ) : list.isError ? (
            <div className={styles.body}>
              <ErrorState error={list.error} onRetry={() => void list.refetch()} />
            </div>
          ) : (
            <div className={styles.body}>
              <Skeleton lines={6} height="20px" />
            </div>
          )}
        </section>
      )}
    </>
  );
}

interface ToolbarProps {
  search: ListSearch;
  total: number | undefined;
  page: PurchaseRequestListPage | undefined;
  onChange: (patch: Partial<ListSearch>, replace?: boolean) => void;
}

/** The toolbar with the PR type options of Settings and the people to choose owners from. */
function Toolbar({ search, total, page, onChange }: ToolbarProps) {
  const form = useAttributeForm('purchaseRequest');
  const people = usePeople();
  const typeAttribute = form.data?.sections
    .flatMap((section) => section.attributes)
    .find((attribute) => attribute.label === 'PR type');
  const types: Choice[] = (typeAttribute?.options ?? []).map((option) => ({
    value: option.id,
    label: option.label,
  }));
  const owners: Choice[] = (people.data?.items ?? []).map((person) => ({
    value: person.id,
    label: person.name,
  }));
  const vendors: Choice[] = (page?.vendors ?? [])
    .filter((group) => group.vendor !== '')
    .map((group) => ({ value: group.vendor, label: group.label }));
  return (
    <PurchaseRequestFilters
      search={search}
      onChange={onChange}
      total={total}
      types={types}
      vendors={vendors}
      owners={owners}
    />
  );
}

interface ResultsProps {
  page: PurchaseRequestListPage;
  search: ListSearch;
  onChange: (patch: Partial<ListSearch>) => void;
}

function Results({ page, search, onChange }: ResultsProps) {
  if (page.items.length > 0) {
    return (
      <>
        <VendorGroupTable page={page} />
        <Pager page={page.page} total={page.total} onPage={(next) => onChange({ page: next })} />
      </>
    );
  }
  if (listFiltersInUse(search)) {
    return (
      <EmptyState
        icon={<SearchX />}
        title={NO_MATCH}
        action={
          <Button onClick={() => onChange(clearedListSearch(search))}>{CLEAR_FILTERS}</Button>
        }
      />
    );
  }
  return (
    <EmptyState
      icon={<FileText />}
      title={search.waiting ? NOTHING_WAITING : NO_PURCHASE_REQUESTS}
    />
  );
}

function Pager({
  page,
  total,
  onPage,
}: {
  page: number;
  total: number;
  onPage: (page: number) => void;
}) {
  const pages = Math.max(1, Math.ceil(total / LIST_PAGE_SIZE));
  if (pages === 1) {
    return null;
  }
  return (
    <nav className={styles.pager} aria-label="Pages">
      <p className={styles.range} aria-live="polite">
        {pageText(page, pages)}
      </p>
      <Button
        size="sm"
        icon={<ChevronLeft />}
        disabled={page <= 1}
        onClick={() => onPage(page - 1)}
      >
        Previous
      </Button>
      <Button size="sm" disabled={page >= pages} onClick={() => onPage(page + 1)}>
        Next
        <ChevronRight aria-hidden="true" className={styles.after} />
      </Button>
    </nav>
  );
}
