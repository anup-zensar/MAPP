import { zodResolver } from '@hookform/resolvers/zod';
import { useState } from 'react';
import { useForm, useWatch } from 'react-hook-form';
import { ChoiceField } from '@/ui/ChoiceField/ChoiceField';
import { FormField } from '@/ui/FormField/FormField';
import { FormPanel } from '@/ui/FormPanel/FormPanel';
import { TextField } from '@/ui/TextField/TextField';
import { useToast } from '@/ui/Toast/useToast';
import { CHECK_ON_SAVE, useFormProblems } from '@/ui/useFormProblems';
import type { SaveFailure } from '@/ui/FormPanel/problem-of';
import { useFormatters } from '@/format/useFormatters';
import styles from '@/ui/settings-table.module.css';
import { ReadOnlyField } from '../ReadOnlyField';
import { saveFailureOf } from '../save-failure';
import {
  fetchFinanceBudgetItem,
  useCreateFinanceBudgetItem,
  useFinanceBudgetPage,
  useUpdateFinanceBudgetItem,
} from './api';
import type { FinanceBudgetItem } from './api';
import { yearLock } from './finance-item-actions';
import { changedFields, createBody, startValues } from './item-values';
import { financeBudgetItemSchema } from './schemas';
import type { FinanceBudgetItemValues } from './schemas';

const FIELDS = ['year', 'budgetName', 'name', 'code'] as const;

export interface FinanceBudgetItemFormProps {
  /** The item to change; without one, the form adds a new item. */
  item?: FinanceBudgetItem;
  /** The budget year a new item starts with: the year shown on the page. */
  year: number;
  /** The budget years that can be chosen. */
  years: number[];
  onClose: () => void;
}

/**
 * New finance budget item and Edit finance budget item (FR-014) in a side panel: the budget year
 * (locked once saved), the budget name with the year's budget names offered while typing, the
 * item name and the code. Items with the same year and budget name form one finance budget, so a
 * new budget name makes a new budget. Every problem shows at once; a conflict keeps what was typed.
 */
export function FinanceBudgetItemForm({ item, year, years, onClose }: FinanceBudgetItemFormProps) {
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const createItem = useCreateFinanceBudgetItem();
  const updateItem = useUpdateFinanceBudgetItem();
  // The edit-form rule: the item is copied into the form once, when the panel opens.
  const [start] = useState(() => startValues(item, year));
  const form = useForm<FinanceBudgetItemValues>({
    resolver: zodResolver(financeBudgetItemSchema),
    defaultValues: start,
    ...CHECK_ON_SAVE,
  });
  const { problems, onInvalid, showServerProblems, clearProblems } = useFormProblems(form, FIELDS);
  const [version, setVersion] = useState(item?.version ?? 0);
  const [saving, setSaving] = useState(false);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  const chosenYear = useWatch({ control: form.control, name: 'year' });
  // The budget names already used in the chosen year, offered while typing (FR-014).
  const yearBudgets = useFinanceBudgetPage(Number(chosenYear));
  const budgetNames =
    yearBudgets.data && !yearBudgets.isPlaceholderData
      ? yearBudgets.data.items.map((budget) => budget.name)
      : [];

  async function save(values: FinanceBudgetItemValues) {
    clearProblems();
    setSaving(true);
    try {
      if (!item) {
        const created = await createItem.mutateAsync(createBody(values));
        show({ message: `Finance budget item '${created.name}' created` });
        onClose();
        return;
      }
      const body = changedFields(values, start, version);
      if (!body) {
        onClose();
        return;
      }
      const saved = await updateItem.mutateAsync({ id: item.id, body });
      show({ message: `Finance budget item '${saved.name}' saved` });
      onClose();
    } catch (error) {
      await showFailure(error);
    } finally {
      setSaving(false);
    }
  }

  async function showFailure(error: unknown) {
    const view = saveFailureOf(error, 'financeBudgetItem', dateFormat);
    showServerProblems(view.problems);
    setFailure(view.failure);
    if (view.conflict && item) {
      // Typed values stay; the next Save replaces the other person's change on purpose.
      const latest = await fetchFinanceBudgetItem(item.id).catch(() => undefined);
      if (latest) {
        setVersion(latest.version);
      }
    }
  }

  const submit = form.handleSubmit(save, onInvalid);

  return (
    <FormPanel
      title={item ? 'Edit finance budget item' : 'New finance budget item'}
      isDirty={form.formState.isDirty}
      saving={saving}
      problems={problems}
      failure={failure}
      onSave={() => void submit()}
      onClose={onClose}
    >
      <div className={styles.formFields}>
        {item ? (
          <ReadOnlyField label="Budget year" value={String(item.year)}>
            {yearLock(item)}
          </ReadOnlyField>
        ) : (
          <FormField
            control={form.control}
            name="year"
            render={(field) => (
              <ChoiceField
                label="Budget year"
                marking="Required"
                options={years.map((each) => ({
                  id: String(each),
                  label: String(each),
                  active: true,
                }))}
                {...field}
              />
            )}
          />
        )}
        <FormField
          control={form.control}
          name="budgetName"
          render={(field) => (
            <TextField
              label="Budget name"
              marking="Required"
              hint="Items with the same year and budget name form one finance budget."
              maxLength={120}
              suggestions={budgetNames}
              {...field}
            />
          )}
        />
        <FormField
          control={form.control}
          name="name"
          render={(field) => (
            <TextField label="Item name" marking="Required" maxLength={120} {...field} />
          )}
        />
        <FormField
          control={form.control}
          name="code"
          render={(field) => <TextField label="Code" maxLength={40} {...field} />}
        />
      </div>
    </FormPanel>
  );
}
