import { useState } from 'react';
import { ChoiceField } from '@/ui/ChoiceField/ChoiceField';
import { FormPanel } from '@/ui/FormPanel/FormPanel';
import { problemOf } from '@/ui/FormPanel/problemOf';
import { LockedNote } from '@/ui/LockedNote/LockedNote';
import { MoneyField } from '@/ui/MoneyField/MoneyField';
import { TextAreaField } from '@/ui/TextAreaField/TextAreaField';
import { TextField } from '@/ui/TextField/TextField';
import { useToast } from '@/ui/Toast/useToast';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import type { SaveFailure } from '@/ui/FormPanel/problemOf';
import { PersonField } from '@/features/people';
import { formatAmount, parseMoneyInput } from '@/format/decimal';
import { useFormatters } from '@/format/useFormatters';
import {
  fetchGlobalBudget,
  useCreateGlobalBudget,
  useUpdateGlobalBudget,
} from './globalBudgetsApi';
import type { GlobalBudget, GlobalBudgetCreate, GlobalBudgetUpdate } from './globalBudgetsApi';
import { toasts, yearChoices } from './globalBudgetsText';
import { saveFailureOf } from './saveFailure';
import styles from './GlobalBudgetForm.module.css';

const AMOUNT_PROBLEM = 'Enter an amount of 0 or more.';
const YEAR_LOCKED = "The year can't change once the global budget is saved.";

export interface GlobalBudgetFormProps {
  /** The global budget to change; without one, the form creates a new global budget. */
  budget?: GlobalBudget;
  onClose: () => void;
  /** After a create: usually opens the new global budget's page. */
  onCreated?: (created: GlobalBudget) => void;
}

interface Values {
  name: string;
  year: string;
  amount: string;
  ownerId: string;
  description: string;
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
  const [start] = useState<Values>(() => ({
    name: budget?.name ?? '',
    year: String(budget?.year ?? currentYear),
    amount: budget ? formatAmount(budget.amount.amount) : '',
    ownerId: budget?.owner?.id ?? '',
    description: budget?.description ?? '',
  }));
  const [values, setValues] = useState(start);
  const [version, setVersion] = useState(budget?.version ?? 0);
  const [saving, setSaving] = useState(false);
  const [problems, setProblems] = useState<FieldProblem[]>([]);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  const yearLock = budget?.locks.find((lock) => lock.field === 'year')?.reason ?? YEAR_LOCKED;

  const set = (field: keyof Values) => (value: string) =>
    setValues((previous) => ({ ...previous, [field]: value }));

  async function save() {
    const amount = parseMoneyInput(values.amount);
    if (!amount || amount.amount.startsWith('-')) {
      setProblems([{ field: 'amount', message: AMOUNT_PROBLEM }]);
      setFailure(null);
      return;
    }
    setSaving(true);
    setProblems([]);
    setFailure(null);
    try {
      if (!budget) {
        const body: GlobalBudgetCreate = {
          name: values.name.trim(),
          year: Number(values.year),
          amount: { amount: amount.amount, currency: 'USD' },
          ownerId: values.ownerId,
          description: values.description,
        };
        const created = await create.mutateAsync(body);
        show({ message: toasts.created(created.name) });
        onClose();
        onCreated?.(created);
        return;
      }
      const body = changedFields(budget, values, amount.amount, version);
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
      setProblems(view.problems);
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

  const sameAsStart = (Object.keys(start) as (keyof Values)[]).every(
    (key) => values[key] === start[key],
  );

  return (
    <FormPanel
      title={budget ? 'Edit global budget' : 'New global budget'}
      description="Money for one year, shared out to market plans. Amounts are in USD."
      isDirty={!sameAsStart}
      saving={saving}
      problems={problems}
      failure={failure}
      onSave={() => void save()}
      onClose={onClose}
    >
      <div className={styles.fields}>
        <TextField
          label="Name"
          name="name"
          marking="Required"
          maxLength={120}
          value={values.name}
          onChange={set('name')}
          error={problemOf(problems, 'name')}
        />
        {budget ? (
          <div className={styles.readOnly}>
            <span className={styles.readOnlyLabel}>Year</span>
            <span className={styles.readOnlyValue}>{budget.year}</span>
            <LockedNote>{yearLock}</LockedNote>
          </div>
        ) : (
          <ChoiceField
            label="Year"
            name="year"
            marking="Required"
            hint="The year can't change once the global budget is saved."
            options={yearChoices(currentYear).map((year) => ({
              id: String(year),
              label: String(year),
              active: true,
            }))}
            value={values.year}
            onChange={set('year')}
            error={problemOf(problems, 'year')}
          />
        )}
        <MoneyField
          label="Amount"
          name="amount"
          marking="Required"
          currency="USD"
          value={values.amount}
          onChange={set('amount')}
          error={problemOf(problems, 'amount')}
        />
        <PersonField
          label="Owner"
          name="ownerId"
          marking="Required"
          hint="A person who can manage global budgets for all markets."
          value={values.ownerId}
          onChange={set('ownerId')}
          error={problemOf(problems, 'ownerId')}
        />
        <TextAreaField
          label="Description"
          name="description"
          maxLength={2000}
          value={values.description}
          onChange={set('description')}
          error={problemOf(problems, 'description')}
        />
      </div>
    </FormPanel>
  );
}

/** The fields people changed, with the version (D-203); null when nothing changed. */
function changedFields(
  budget: GlobalBudget,
  values: Values,
  amount: string,
  version: number,
): GlobalBudgetUpdate | null {
  const body: GlobalBudgetUpdate = { version };
  if (values.name.trim() !== budget.name) {
    body.name = values.name.trim();
  }
  if (amount !== budget.amount.amount) {
    body.amount = { amount, currency: 'USD' };
  }
  if (values.ownerId !== (budget.owner?.id ?? '')) {
    body.ownerId = values.ownerId;
  }
  if (values.description !== budget.description) {
    body.description = values.description;
  }
  return Object.keys(body).length > 1 ? body : null;
}
