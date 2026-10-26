import { zodResolver } from '@hookform/resolvers/zod';
import { useState } from 'react';
import { useForm, useWatch } from 'react-hook-form';
import { DecimalField } from '@/ui/DecimalField/DecimalField';
import { FormField } from '@/ui/FormField/FormField';
import { FormPanel } from '@/ui/FormPanel/FormPanel';
import { TextField } from '@/ui/TextField/TextField';
import { useToast } from '@/ui/Toast/useToast';
import { CHECK_ON_SAVE, useFormProblems } from '@/ui/useFormProblems';
import type { SaveFailure } from '@/ui/FormPanel/problem-of';
import { useFormatters } from '@/format/useFormatters';
import styles from '@/ui/settings-table.module.css';
import { ImpactDialog } from '../ImpactDialog';
import { saveFailureOf } from '../save-failure';
import { checkCurrencyImpact, fetchCurrency, useCreateCurrency, useUpdateCurrency } from './api';
import type { ChangeImpact, Currency, CurrencyUpdate } from './api';
import { lockOf, rateLine } from './currency-actions';
import { changedFields, createBody, startValues } from './currency-values';
import { liveRate } from './rate-input';
import { rateSourceLine } from './rate-words';
import { ReadOnlyField } from '../ReadOnlyField';
import { currencySchema } from './schemas';
import type { CurrencyValues } from './schemas';

/** A changed rate becomes today's dated rate (016 FR-004). */
const TODAYS_RATE_HINT = "A new rate is saved as today's rate; the rate history keeps the others.";

/** A new currency's rate is its first dated rate (016 FR-003). */
const FIRST_RATE_HINT = "Saved as the currency's first rate, dated today.";

/** The code lock's reason before the API has given one (a new currency is not saved yet). */
const CODE_LOCKED = "The code can't change once the currency is saved.";

const FIELDS = ['code', 'name', 'symbol', 'rate'] as const;

export interface CurrencyFormProps {
  /** The currency to change; without one, the form adds a new currency. */
  currency?: Currency;
  onClose: () => void;
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
  const form = useForm<CurrencyValues>({
    resolver: zodResolver(currencySchema),
    // The edit-form rule: the currency is copied into the form once, when the panel opens.
    defaultValues: startValues(currency),
    ...CHECK_ON_SAVE,
  });
  const { problems, onInvalid, showServerProblems, clearProblems } = useFormProblems(form, FIELDS);
  const [version, setVersion] = useState(currency?.version ?? 0);
  const [saving, setSaving] = useState(false);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  const [pendingImpact, setPendingImpact] = useState<PendingImpact | null>(null);
  const [typedCode, typedRate] = useWatch({ control: form.control, name: ['code', 'rate'] });
  const rateLock = currency ? lockOf(currency, 'rate') : undefined;

  async function save(values: CurrencyValues) {
    clearProblems();
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
    showServerProblems(view.problems);
    setFailure(view.failure);
    if (view.conflict && currency) {
      // Typed values stay; the next Save replaces the other person's change on purpose.
      const latest = await fetchCurrency(currency.code).catch(() => undefined);
      if (latest) {
        setVersion(latest.version);
      }
    }
  }

  const submit = form.handleSubmit(save, onInvalid);
  const code = (currency?.code ?? typedCode.trim()).toUpperCase();

  return (
    <>
      <FormPanel
        title={currency ? 'Edit currency' : 'New currency'}
        isDirty={form.formState.isDirty}
        saving={saving}
        problems={problems}
        failure={failure}
        onSave={() => void submit()}
        onClose={onClose}
      >
        <div className={styles.formFields}>
          {currency ? (
            <ReadOnlyField label="Code" value={currency.code}>
              {lockOf(currency, 'code') ?? CODE_LOCKED}
            </ReadOnlyField>
          ) : (
            <FormField
              control={form.control}
              name="code"
              render={(field) => (
                <TextField
                  label="Code"
                  marking="Required"
                  hint="Three letters, as in ISO 4217 (EUR, JPY)."
                  {...field}
                />
              )}
            />
          )}
          <FormField
            control={form.control}
            name="name"
            render={(field) => <TextField label="Name" marking="Required" {...field} />}
          />
          <FormField
            control={form.control}
            name="symbol"
            render={(field) => <TextField label="Symbol" {...field} />}
          />
          {rateLock && currency ? (
            <ReadOnlyField
              label="Rate (units per 1 USD)"
              value={rateLine(currency.code, currency.rate)}
            >
              {rateLock}
            </ReadOnlyField>
          ) : (
            <FormField
              control={form.control}
              name="rate"
              render={(field) => (
                <DecimalField
                  label="Rate (units per 1 USD)"
                  marking="Required"
                  decimals={6}
                  suffix="per USD"
                  hint={currency ? TODAYS_RATE_HINT : FIRST_RATE_HINT}
                  note={liveRate(code, typedRate)}
                  {...field}
                />
              )}
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
