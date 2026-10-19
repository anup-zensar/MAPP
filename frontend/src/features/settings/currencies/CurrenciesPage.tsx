import { Plus } from 'lucide-react';
import { useState } from 'react';
import { useCurrencies } from '@/api/settingsQueries';
import {
  ActionsMenu,
  Breadcrumb,
  Button,
  Card,
  ErrorState,
  PageHeader,
  SearchField,
  SegmentedFilter,
  Skeleton,
  StatusChip,
  useToast,
} from '@/components';
import { localDate } from '@/lib/format';
import { useFormatters } from '@/lib/useFormatters';
import { usePageTitle } from '@/lib/usePageTitle';
import { SettingDeleteDialog } from '../shared/SettingDeleteDialog';
import styles from '../shared/SettingsTable.module.css';
import { filterRecords, useListFilters } from '../shared/useListFilters';
import type { ListFilters } from '../shared/useListFilters';
import { useSettingStatus } from '../shared/useSettingStatus';
import { CURRENCY_LISTS, setCurrencyStatus, useDeleteCurrency } from './currenciesApi';
import type { Currency } from './currenciesApi';
import { inverseLine, moreActions, rateLine } from './currencyActions';
import type { CurrencyActions } from './currencyActions';
import { CurrencyDetails } from './CurrencyDetails';
import { CurrencyForm } from './CurrencyForm';

/** The side-panel form that is open: a new currency, or one being changed. */
type OpenForm = { kind: 'new' } | { kind: 'edit'; currency: Currency };

/**
 * Currencies (feature 002, US4, FR-028 – FR-030): the currencies line items and people can use,
 * with their rate per 1 USD both ways, as the standard CRUD pattern — search and status filter in
 * the address, a table, details in a side panel (`?id=EUR`), New currency and Edit in a side-panel
 * form, Set inactive / Reactivate and Delete (unused currencies only, with Undo). USD's rate,
 * status and delete are locked with their reason.
 */
export function CurrenciesPage() {
  usePageTitle('Currencies');
  const filters = useListFilters();
  const currencies = useCurrencies();
  const deleteCurrency = useDeleteCurrency();
  const { show } = useToast();
  const [form, setForm] = useState<OpenForm | null>(null);
  const [deleting, setDeleting] = useState<Currency | null>(null);
  const changeStatus = useSettingStatus({
    kind: 'currency',
    request: setCurrencyStatus,
    invalidate: CURRENCY_LISTS,
  });

  const target = (currency: Currency) => ({
    id: currency.code,
    name: currency.name,
    version: currency.version,
  });
  const actions: CurrencyActions = {
    view: (currency) => filters.setOpenId(currency.code),
    edit: (currency) => setForm({ kind: 'edit', currency }),
    setActive: (currency, active) => void changeStatus(target(currency), active),
    remove: (currency) => setDeleting(currency),
    explainLock: (reason) => show({ message: reason }),
  };
  const newCurrency = () => setForm({ kind: 'new' });
  // The details wait for the list; a currency that is not in it any more says so.
  const openCurrency =
    filters.openId && currencies.data
      ? (currencies.data.items.find((currency) => currency.code === filters.openId) ?? null)
      : undefined;

  return (
    <>
      <Breadcrumb items={[{ label: 'Settings', to: '/settings' }, { label: 'Currencies' }]} />
      <PageHeader
        eyebrow="Settings"
        title="Currencies"
        description="Currencies for line items and display, with their rate per 1 USD."
        actions={
          <Button variant="primary" icon={<Plus />} onClick={newCurrency}>
            New currency
          </Button>
        }
      />
      <CurrenciesCard currencies={currencies} filters={filters} actions={actions} />

      {form && (
        <CurrencyForm
          currency={form.kind === 'edit' ? form.currency : undefined}
          onClose={() => setForm(null)}
        />
      )}
      {!form && openCurrency !== undefined && (
        <CurrencyDetails
          currency={openCurrency}
          actions={actions}
          onClose={() => filters.setOpenId('')}
        />
      )}
      {deleting && (
        <SettingDeleteDialog
          kind="currency"
          name={deleting.name}
          usage={deleting.usage}
          active={deleting.active}
          remove={() => deleteCurrency.mutateAsync(deleting.code)}
          setInactive={() => changeStatus(target(deleting), false)}
          invalidate={CURRENCY_LISTS}
          onDeleted={() => {
            if (filters.openId === deleting.code) {
              filters.setOpenId('');
            }
          }}
          onClose={() => setDeleting(null)}
        />
      )}
    </>
  );
}

interface CurrenciesCardProps {
  currencies: ReturnType<typeof useCurrencies>;
  filters: ListFilters;
  actions: CurrencyActions;
}

/** The toolbar and the table, or what to show instead: loading or failure. */
function CurrenciesCard({ currencies, filters, actions }: CurrenciesCardProps) {
  if (!currencies.data) {
    return (
      <Card>
        {currencies.isError ? (
          <ErrorState error={currencies.error} onRetry={() => void currencies.refetch()} />
        ) : (
          <Skeleton lines={5} height="18px" />
        )}
      </Card>
    );
  }

  // USD always exists (FR-028), so the list is never empty.
  const all = currencies.data.items;
  const { shown, counts } = filterRecords(
    all,
    filters,
    (currency) => `${currency.code} ${currency.name}`,
  );

  return (
    <Card>
      <div className={styles.toolbar}>
        <SearchField
          label="Search currencies"
          placeholder="Search code or name"
          value={filters.q}
          onChange={filters.setQ}
        />
        <SegmentedFilter
          label="Status"
          options={[
            { value: 'active', label: 'Active', count: counts.active },
            { value: 'inactive', label: 'Inactive', count: counts.inactive },
            { value: 'all', label: 'All', count: counts.all },
          ]}
          value={filters.status}
          onChange={filters.setStatus}
        />
        <p className={styles.resultCount} aria-live="polite">
          {`${shown.length} of ${all.length}`}
        </p>
      </div>
      {shown.length === 0 ? (
        <div className={styles.noMatch}>
          <p>Nothing matches these filters</p>
          <Button onClick={filters.clear}>Clear filters</Button>
        </div>
      ) : (
        <div className={styles.tableScroll}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th scope="col">Code</th>
                <th scope="col">Name</th>
                <th scope="col">Symbol</th>
                <th scope="col">Rate</th>
                <th scope="col">Used by</th>
                <th scope="col">Rate last changed</th>
                <th scope="col">Status</th>
                <th scope="col">
                  <span className="visually-hidden">Actions</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {shown.map((currency) => (
                <CurrencyRow key={currency.code} currency={currency} actions={actions} />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </Card>
  );
}

function CurrencyRow({ currency, actions }: { currency: Currency; actions: CurrencyActions }) {
  const { date } = useFormatters();
  return (
    <tr>
      <th scope="row">
        <button type="button" className={styles.nameButton} onClick={() => actions.view(currency)}>
          {currency.code}
        </button>
      </th>
      <td>{currency.name}</td>
      <td>{currency.symbol || '—'}</td>
      <td className={styles.numeric}>
        {rateLine(currency.code, currency.rate)}
        <span className={styles.secondLine}>
          {inverseLine(currency.code, currency.inverseRate)}
        </span>
      </td>
      <td>{currency.usage.summary}</td>
      <td className={styles.numeric}>{date(localDate(currency.rateChangedAt))}</td>
      <td>
        <StatusChip tone={currency.active ? 'positive' : 'neutral'}>
          {currency.active ? 'Active' : 'Inactive'}
        </StatusChip>
      </td>
      <td className={styles.actionsCell}>
        <ActionsMenu
          label={currency.code}
          items={[
            { label: 'View', onSelect: () => actions.view(currency) },
            { label: 'Edit', onSelect: () => actions.edit(currency) },
            ...moreActions(currency, actions),
          ]}
        />
      </td>
    </tr>
  );
}
