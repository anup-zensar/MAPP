import { Plus, Upload } from 'lucide-react';
import { useState } from 'react';
import { useCurrencies } from '@/api/settingsQueries';
import {
  ActionsMenu,
  Button,
  Card,
  ErrorState,
  LinkButton,
  PageHeader,
  Skeleton,
  StatusChip,
  useToast,
} from '@/components';
import { useFormatters } from '@/lib/useFormatters';
import { usePageTitle } from '@/lib/usePageTitle';
import { ListToolbar } from '../shared/ListToolbar';
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
import { CURRENCY_IMPORT_PATH } from './currencyPaths';
import { rateDateOf, SOURCE_WORDS } from './rateWords';
import styles016 from './rates.module.css';
import { RecentImports } from './RecentImports';
import { CanChange, SettingsViewOnlyNote } from '../shared/CanChange';
import { useCanChange } from '../shared/useCanChange';

const CURRENCY_NOUN = { one: 'currency', many: 'currencies' };

/** The side-panel form that is open: a new currency, or one being changed. */
type OpenForm = { kind: 'new' } | { kind: 'edit'; currency: Currency };

/**
 * Currencies (feature 002, US4, FR-028 – FR-030; 016 US1): the currencies line items, plans and
 * people can use, with their current rate per 1 USD both ways and its date, **Import GSAP rates**
 * and the recent imports (016 FR-006, FR-007), as the standard CRUD pattern — search and status filter in
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
  const status = useSettingStatus({
    kind: 'currency',
    request: setCurrencyStatus,
    invalidate: CURRENCY_LISTS,
  });

  const target = (currency: Currency) => ({
    id: currency.code,
    name: currency.name,
    version: currency.version,
    usage: currency.usage,
  });
  const actions: CurrencyActions = {
    view: (currency) => filters.setOpenId(currency.code),
    edit: (currency) => setForm({ kind: 'edit', currency }),
    setActive: (currency, active) =>
      active ? status.reactivate(target(currency)) : status.setInactive(target(currency)),
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
      <PageHeader
        breadcrumb={[{ label: 'Settings', to: '/settings' }, { label: 'Currencies' }]}
        title="Currencies"
        description="Currencies for line items, plans and display, with their dated rates per 1 USD."
        actions={
          <CanChange permission="exchangeRates.manage">
            <>
              <LinkButton icon={<Upload />} to={CURRENCY_IMPORT_PATH}>
                Import GSAP rates
              </LinkButton>
              <Button variant="primary" icon={<Plus />} onClick={newCurrency}>
                New currency
              </Button>
            </>
          </CanChange>
        }
      />
      <SettingsViewOnlyNote permission="exchangeRates.manage" />
      <div className={styles016.stack}>
        <CurrenciesCard currencies={currencies} filters={filters} actions={actions} />
        <RecentImports />
      </div>

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
          setInactive={() => status.applyInactive(target(deleting))}
          invalidate={CURRENCY_LISTS}
          onDeleted={() => {
            if (filters.openId === deleting.code) {
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
      <ListToolbar
        search={{
          label: 'Search currencies',
          placeholder: 'Search code or name',
          value: filters.q,
          onChange: filters.setQ,
        }}
        status={{ value: filters.status, onChange: filters.setStatus, counts }}
        count={{ shown: shown.length, total: all.length, noun: CURRENCY_NOUN }}
      />
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
                <th scope="col" className={styles.number}>
                  Rate
                </th>
                <th scope="col">Used by</th>
                <th scope="col">Rate of</th>
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
  const canChange = useCanChange('exchangeRates.manage');
  const { date } = useFormatters();
  return (
    <tr>
      <th scope="row">
        <button type="button" className={styles.nameButton} onClick={() => actions.view(currency)}>
          {currency.code}
        </button>
      </th>
      <td className={styles.nowrap}>{currency.name}</td>
      <td>{currency.symbol || '—'}</td>
      <td className={styles.number}>
        {rateLine(currency.code, currency.rate)}
        <span className={styles.secondLine}>
          {inverseLine(currency.code, currency.inverseRate)}
        </span>
      </td>
      <td>{currency.usage.summary}</td>
      <td className={styles.nowrap}>
        {date(rateDateOf(currency))}
        {currency.rateSource && (
          <span className={styles.secondLine}>{SOURCE_WORDS[currency.rateSource]}</span>
        )}
      </td>
      <td className={styles.nowrap}>
        <StatusChip tone={currency.active ? 'positive' : 'neutral'}>
          {currency.active ? 'Active' : 'Inactive'}
        </StatusChip>
      </td>
      <td className={styles.actionsCell}>
        <ActionsMenu
          label={currency.code}
          items={[
            { label: 'View', onSelect: () => actions.view(currency) },
            ...(canChange
              ? [
                  { label: 'Edit', onSelect: () => actions.edit(currency) },
                  ...moreActions(currency, actions),
                ]
              : []),
          ]}
        />
      </td>
    </tr>
  );
}
