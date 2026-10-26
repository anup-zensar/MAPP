import { zodResolver } from '@hookform/resolvers/zod';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { ChoiceField } from '@/ui/ChoiceField/ChoiceField';
import { FormField } from '@/ui/FormField/FormField';
import { FormPanel } from '@/ui/FormPanel/FormPanel';
import { LockedNote } from '@/ui/LockedNote/LockedNote';
import { MoneyField } from '@/ui/MoneyField/MoneyField';
import { TextAreaField } from '@/ui/TextAreaField/TextAreaField';
import { TextField } from '@/ui/TextField/TextField';
import { useToast } from '@/ui/Toast/useToast';
import { CHECK_ON_SAVE, useFormProblems } from '@/ui/useFormProblems';
import type { SaveFailure } from '@/ui/FormPanel/problem-of';
import { PersonField } from '@/features/people';
import { useFormatters } from '@/format/useFormatters';
import { fetchGlobalBudget, useCreateGlobalBudget, useUpdateGlobalBudget } from './api';
import type { GlobalBudget } from './api';
import { toasts, yearChoices } from './global-budgets-text';
import { saveFailureOf } from './save-failure';
import {
  amountOf,
  changedFields,
  globalBudgetSchema,
  globalBudgetValuesOf,
  newGlobalBudget,
} from './schemas';
import type { GlobalBudgetValues } from './schemas';
import styles from './GlobalBudgetForm.module.css';

const FIELDS = ['name', 'year', 'amount', 'ownerId', 'description'] as const;
const YEAR_LOCKED = "The year can't change once the global budget is saved.";

export interface GlobalBudgetFormProps {
  /** The global budget to change; without one, the form creates a new global budget. */
  budget?: GlobalBudget;
  onClose: () => void;
  /** After a create: usually opens the new global budget's page. */
  onCreated?: (created: GlobalBudget) => void;
}

/**
 * New global budget and Edit global budget (014 FR-029) in a side panel: name, year (fixed once
 * saved), amount in USD, owner and description. Field problems come from the API, all at once.
 */
export function GlobalBudgetForm({ budget, onClose, onCreated }: GlobalBudgetFormProps) {
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const create = useCreateGlobalBudget();
  const update = useUpdateGlobalBudget();
  const currentYear = new Date().getFullYear();
  const form = useForm<GlobalBudgetValues>({
    resolver: zodResolver(globalBudgetSchema),
    defaultValues: globalBudgetValuesOf(budget, currentYear),
    ...CHECK_ON_SAVE,
  });
  const { problems, onInvalid, showServerProblems, clearProblems } = useFormProblems(form, FIELDS);
  const [version, setVersion] = useState(budget?.version ?? 0);
  const [saving, setSaving] = useState(false);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  const yearLock = budget?.locks.find((lock) => lock.field === 'year')?.reason ?? YEAR_LOCKED;

  async function save(values: GlobalBudgetValues) {
    const amount = amountOf(values.amount);
    if (amount === null) {
      return;
    }
    setSaving(true);
    clearProblems();
    setFailure(null);
    try {
      if (!budget) {
        const created = await create.mutateAsync(newGlobalBudget(values, amount));
        show({ message: toasts.created(created.name) });
        onClose();
        onCreated?.(created);
        return;
      }
      const body = changedFields(budget, values, amount, version);
      if (!body) {
        onClose();
        return;
      }
      const saved = await update.mutateAsync({ id: budget.id, body });
      show({ message: toasts.saved(saved.name) });
      onClose();
    } catch (error) {
      const view = saveFailureOf(error, 'global budget', dateFormat);
      if (view.sessionLost) {
        return;
      }
      showServerProblems(view.problems);
      setFailure(view.failure);
      if (view.conflict && budget) {
        // Typed values stay; the next Save replaces the other person's change on purpose.
        const latest = await fetchGlobalBudget(budget.id).catch(() => undefined);
        if (latest) {
          setVersion(latest.version);
        }
      }
    } finally {
      setSaving(false);
    }
  }

  function showTypedProblems() {
    onInvalid();
    setFailure(null);
  }

  return (
    <FormPanel
      title={budget ? 'Edit global budget' : 'New global budget'}
      description="Money for one year, shared out to market plans. Amounts are in USD."
      isDirty={form.formState.isDirty}
      saving={saving}
      problems={problems}
      failure={failure}
      onSave={() => void form.handleSubmit(save, showTypedProblems)()}
      onClose={onClose}
    >
      <div className={styles.fields}>
        <FormField
          control={form.control}
          name="name"
          render={(field) => (
            <TextField label="Name" marking="Required" maxLength={120} {...field} />
          )}
        />
        {budget ? (
          <div className={styles.readOnly}>
            <span className={styles.readOnlyLabel}>Year</span>
            <span className={styles.readOnlyValue}>{budget.year}</span>
            <LockedNote>{yearLock}</LockedNote>
          </div>
        ) : (
          <FormField
            control={form.control}
            name="year"
            render={(field) => (
              <ChoiceField
                label="Year"
                marking="Required"
                hint="The year can't change once the global budget is saved."
                options={yearChoices(currentYear).map((year) => ({
                  id: String(year),
                  label: String(year),
                  active: true,
                }))}
                {...field}
              />
            )}
          />
        )}
        <FormField
          control={form.control}
          name="amount"
          render={(field) => (
            <MoneyField label="Amount" marking="Required" currency="USD" {...field} />
          )}
        />
        <FormField
          control={form.control}
          name="ownerId"
          render={(field) => (
            <PersonField
              label="Owner"
              marking="Required"
              hint="A person who can manage global budgets for all markets."
              {...field}
            />
          )}
        />
        <FormField
          control={form.control}
          name="description"
          render={(field) => <TextAreaField label="Description" maxLength={2000} {...field} />}
        />
      </div>
    </FormPanel>
  );
}
