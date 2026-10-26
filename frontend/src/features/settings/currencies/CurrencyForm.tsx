import { useState } from 'react';
import { DecimalField } from '@/ui/DecimalField/DecimalField';
import { FormPanel } from '@/ui/FormPanel/FormPanel';
import { problemOf } from '@/ui/FormPanel/problemOf';
import { LockedNote } from '@/ui/LockedNote/LockedNote';
import { TextField } from '@/ui/TextField/TextField';
import { useToast } from '@/ui/Toast/useToast';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import type { SaveFailure } from '@/ui/FormPanel/problemOf';
import { inverseRate, parseDecimalInput } from '@/format/decimal';
import { useFormatters } from '@/format/useFormatters';
import { ImpactDialog } from '../shared/ImpactDialog';
import { saveFailureOf } from '../shared/saveFailure';
import styles from '../shared/SettingsTable.module.css';
import {
  checkCurrencyImpact,
  fetchCurrency,
  useCreateCurrency,
  useUpdateCurrency,
} from './currenciesApi';
import type { ChangeImpact, Currency, CurrencyCreate, CurrencyUpdate } from './currenciesApi';
import { inverseLine, lockOf, rateLine } from './currencyActions';
import { rateSourceLine } from './rateWords';

/** A changed rate becomes today's dated rate (016 FR-004). */
const TODAYS_RATE_HINT = "A new rate is saved as today's rate; the rate history keeps the others.";

/** A new currency's rate is its first dated rate (016 FR-003). */
const FIRST_RATE_HINT = "Saved as the currency's first rate, dated today.";

/** The code lock's reason before the API has given one (a new currency is not saved yet). */
const CODE_LOCKED = "The code can't change once the currency is saved.";

export interface CurrencyFormProps {
  /** The currency to change; without one, the form adds a new currency. */
  currency?: Currency;
  onClose: () => void;
}

/** The typed values, as text, exactly as they are in the fields. */
interface CurrencyValues {
  code: string;
  name: string;
  symbol: string;
  rate: string;
}

/** A save that waits for "Save changes" in the impact dialog. */
interface PendingImpact {
  body: CurrencyUpdate;
  warnings: ChangeImpact['warnings'];
}

/**
 * New currency and Edit currency (FR-028 – FR-030) in a side panel: code (locked once saved), name,
 * symbol and the rate per 1 USD with the live line "1 USD = 0.92 EUR · 1 EUR = 1.086957 USD".
 * USD's rate is locked. A new rate is confirmed with what it recalculates before it is saved.
 */
export function CurrencyForm({ currency, onClose }: CurrencyFormProps) {
  const { show } = useToast();
  const { dateFormat, date } = useFormatters();
  const createCurrency = useCreateCurrency();
  const updateCurrency = useUpdateCurrency();
  // The edit-form rule: the currency is copied into the form once, when the panel opens.
  const [start] = useState<CurrencyValues>(() => ({
    code: currency?.code ?? '',
    name: currency?.name ?? '',
    symbol: currency?.symbol ?? '',
    rate: currency?.rate ?? '',
  }));
  const [values, setValues] = useState(start);
  const [version, setVersion] = useState(currency?.version ?? 0);
  const [saving, setSaving] = useState(false);
  const [problems, setProblems] = useState<FieldProblem[]>([]);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  const [pendingImpact, setPendingImpact] = useState<PendingImpact | null>(null);
  const rateLock = currency ? lockOf(currency, 'rate') : undefined;

  function set<K extends keyof CurrencyValues>(field: K, value: CurrencyValues[K]) {
    setValues((previous) => ({ ...previous, [field]: value }));
  }

  async function save() {
    setSaving(true);
    try {
      if (!currency) {
        const created = await createCurrency.mutateAsync(createBody(values));
        show({ message: `Currency '${created.name}' created` });
        onClose();
        return;
      }
      const body = changedFields(currency, values, version);
      if (!body) {
        onClose();
        return;
      }
      if (body.rate !== undefined) {
        const impact = await checkCurrencyImpact(currency.code, body);
        if (impact.warnings.length > 0) {
          setPendingImpact({ body, warnings: impact.warnings });
          return;
        }
      }
      await saveChanges(currency, body);
    } catch (error) {
      await showFailure(error);
    } finally {
      setSaving(false);
    }
  }

  async function saveChanges(saved: Currency, body: CurrencyUpdate) {
    const updated = await updateCurrency.mutateAsync({ code: saved.code, body });
    show({ message: `Currency '${updated.name}' saved` });
    onClose();
  }

  async function confirmImpact() {
    if (!currency || !pendingImpact) {
      return;
    }
    setSaving(true);
    try {
      await saveChanges(currency, pendingImpact.body);
    } catch (error) {
      await showFailure(error);
    } finally {
      setSaving(false);
      setPendingImpact(null);
    }
  }

  async function showFailure(error: unknown) {
    const view = saveFailureOf(error, 'currency', dateFormat);
    setProblems(view.problems);
    setFailure(view.failure);
    if (view.conflict && currency) {
      // Typed values stay; the next Save replaces the other person's change on purpose.
      const latest = await fetchCurrency(currency.code).catch(() => undefined);
      if (latest) {
        setVersion(latest.version);
      }
    }
  }

  const code = (currency?.code ?? values.code.trim()).toUpperCase();

  return (
    <>
      <FormPanel
        title={currency ? 'Edit currency' : 'New currency'}
        isDirty={!sameValues(values, start)}
        saving={saving}
        problems={problems}
        failure={failure}
        onSave={() => void save()}
        onClose={onClose}
      >
        <div className={styles.formFields}>
          {currency ? (
            <ReadOnlyField label="Code" value={currency.code}>
              {lockOf(currency, 'code') ?? CODE_LOCKED}
            </ReadOnlyField>
          ) : (
            <TextField
              label="Code"
              name="code"
              marking="Required"
              hint="Three letters, as in ISO 4217 (EUR, JPY)."
              value={values.code}
              onChange={(text) => set('code', text)}
              error={problemOf(problems, 'code')}
            />
          )}
          <TextField
            label="Name"
            name="name"
            marking="Required"
            value={values.name}
            onChange={(text) => set('name', text)}
            error={problemOf(problems, 'name')}
          />
          <TextField
            label="Symbol"
            name="symbol"
            value={values.symbol}
            onChange={(text) => set('symbol', text)}
            error={problemOf(problems, 'symbol')}
          />
          {rateLock && currency ? (
            <ReadOnlyField
              label="Rate (units per 1 USD)"
              value={rateLine(currency.code, currency.rate)}
            >
              {rateLock}
            </ReadOnlyField>
          ) : (
            <DecimalField
              label="Rate (units per 1 USD)"
              name="rate"
              marking="Required"
              decimals={6}
              suffix="per USD"
              hint={currency ? TODAYS_RATE_HINT : FIRST_RATE_HINT}
              value={values.rate}
              onChange={(text) => set('rate', text)}
              note={liveRate(code, values.rate)}
              error={problemOf(problems, 'rate')}
            />
          )}
          {currency && <p className={styles.meta}>{rateSourceLine(currency, date)}</p>}
        </div>
      </FormPanel>
      {pendingImpact && currency && (
        <ImpactDialog
          kind="currency"
          name={currency.name}
          warnings={pendingImpact.warnings}
          busy={saving}
          onConfirm={() => void confirmImpact()}
          onCancel={() => setPendingImpact(null)}
        />
      )}
    </>
  );
}

interface ReadOnlyFieldProps {
  label: string;
  value: string;
  /** Why it can't change. */
  children: string;
}

/** A value that can't change, in place of its field, with the lock and the reason. */
function ReadOnlyField({ label, value, children }: ReadOnlyFieldProps) {
  return (
    <div className={styles.readOnly}>
      <span className={styles.readOnlyLabel}>{label}</span>
      <span className={styles.readOnlyValue}>{value}</span>
      <LockedNote>{children}</LockedNote>
    </div>
  );
}

/** "1 USD = 0.92 EUR · 1 EUR = 1.086957 USD" for a typed rate above 0; nothing otherwise. */
function liveRate(code: string, typed: string): string | undefined {
  const rate = parseDecimalInput(typed, 6);
  const inverse = rate === null ? null : inverseRate(rate);
  if (rate === null || inverse === null || code === '') {
    return undefined;
  }
  return `${rateLine(code, rate)} · ${inverseLine(code, inverse)}`;
}

function sameValues(a: CurrencyValues, b: CurrencyValues): boolean {
  return a.code === b.code && a.name === b.name && a.symbol === b.symbol && a.rate === b.rate;
}

/** A typed rate as the API takes it: grouping removed; anything else as typed, for the API to judge. */
function rateText(text: string): string {
  return parseDecimalInput(text, 6) ?? text.trim();
}

function createBody(values: CurrencyValues): CurrencyCreate {
  return {
    code: values.code.trim(),
    name: values.name,
    symbol: values.symbol,
    rate: rateText(values.rate),
  };
}

/** The fields people changed, with the version (D-203); null when nothing changed. */
function changedFields(
  currency: Currency,
  values: CurrencyValues,
  version: number,
): CurrencyUpdate | null {
  const body: CurrencyUpdate = { version };
  if (values.name !== currency.name) {
    body.name = values.name;
  }
  if (values.symbol !== currency.symbol) {
    body.symbol = values.symbol;
  }
  if (!lockOf(currency, 'rate') && !sameRate(values.rate, currency.rate)) {
    body.rate = rateText(values.rate);
  }
  return Object.keys(body).length > 1 ? body : null;
}

/** "0.920" and "0.92" are the same rate. */
function sameRate(typed: string, saved: string): boolean {
  const rate = parseDecimalInput(typed, 6);
  return rate !== null && trimZeros(rate) === trimZeros(saved);
}

function trimZeros(rate: string): string {
  return rate.includes('.') ? rate.replace(/\.?0+$/, '') : rate;
}
