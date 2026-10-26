import { useState } from 'react';
import { ChoiceField } from '@/ui/ChoiceField/ChoiceField';
import { FormPanel } from '@/ui/FormPanel/FormPanel';
import { problemOf } from '@/ui/FormPanel/problemOf';
import { LockedNote } from '@/ui/LockedNote/LockedNote';
import { TextField } from '@/ui/TextField/TextField';
import { useToast } from '@/ui/Toast/useToast';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import type { SaveFailure } from '@/ui/FormPanel/problemOf';
import { useFormatters } from '@/lib/useFormatters';
import { saveFailureOf } from '../shared/saveFailure';
import styles from '../shared/SettingsTable.module.css';
import {
  fetchFinanceBudgetItem,
  useCreateFinanceBudgetItem,
  useFinanceBudgetPage,
  useUpdateFinanceBudgetItem,
} from './financeBudgetsApi';
import type { FinanceBudgetItem, FinanceBudgetItemUpdate } from './financeBudgetsApi';
import { yearLock } from './financeItemActions';

export interface FinanceBudgetItemFormProps {
  /** The item to change; without one, the form adds a new item. */
  item?: FinanceBudgetItem;
  /** The budget year a new item starts with: the year shown on the page. */
  year: number;
  /** The budget years that can be chosen. */
  years: number[];
  onClose: () => void;
}

/** The typed values, as text, exactly as they are in the fields. */
interface ItemValues {
  year: string;
  budgetName: string;
  name: string;
  code: string;
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
  const [start] = useState<ItemValues>(() => ({
    year: String(item?.year ?? year),
    budgetName: item?.budgetName ?? '',
    name: item?.name ?? '',
    code: item?.code ?? '',
  }));
  const [values, setValues] = useState(start);
  const [version, setVersion] = useState(item?.version ?? 0);
  const [saving, setSaving] = useState(false);
  const [problems, setProblems] = useState<FieldProblem[]>([]);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  // The budget names already used in the chosen year, offered while typing (FR-014).
  const yearBudgets = useFinanceBudgetPage(Number(values.year));
  const budgetNames =
    yearBudgets.data && !yearBudgets.isPlaceholderData
      ? yearBudgets.data.items.map((budget) => budget.name)
      : [];

  function set<K extends keyof ItemValues>(field: K, value: ItemValues[K]) {
    setValues((previous) => ({ ...previous, [field]: value }));
  }

  async function save() {
    setSaving(true);
    try {
      if (!item) {
        const created = await createItem.mutateAsync({
          year: Number(values.year),
          budgetName: values.budgetName,
          name: values.name,
          code: values.code,
        });
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
    setProblems(view.problems);
    setFailure(view.failure);
    if (view.conflict && item) {
      // Typed values stay; the next Save replaces the other person's change on purpose.
      const latest = await fetchFinanceBudgetItem(item.id).catch(() => undefined);
      if (latest) {
        setVersion(latest.version);
      }
    }
  }

  return (
    <FormPanel
      title={item ? 'Edit finance budget item' : 'New finance budget item'}
      isDirty={!sameValues(values, start)}
      saving={saving}
      problems={problems}
      failure={failure}
      onSave={() => void save()}
      onClose={onClose}
    >
      <div className={styles.formFields}>
        {item ? (
          <div className={styles.readOnly}>
            <span className={styles.readOnlyLabel}>Budget year</span>
            <span className={styles.readOnlyValue}>{item.year}</span>
            <LockedNote>{yearLock(item)}</LockedNote>
          </div>
        ) : (
          <ChoiceField
            label="Budget year"
            name="year"
            marking="Required"
            options={years.map((each) => ({ id: String(each), label: String(each), active: true }))}
            value={values.year}
            onChange={(next) => set('year', next)}
            error={problemOf(problems, 'year')}
          />
        )}
        <TextField
          label="Budget name"
          name="budgetName"
          marking="Required"
          hint="Items with the same year and budget name form one finance budget."
          maxLength={120}
          suggestions={budgetNames}
          value={values.budgetName}
          onChange={(budgetName) => set('budgetName', budgetName)}
          error={problemOf(problems, 'budgetName')}
        />
        <TextField
          label="Item name"
          name="name"
          marking="Required"
          maxLength={120}
          value={values.name}
          onChange={(name) => set('name', name)}
          error={problemOf(problems, 'name')}
        />
        <TextField
          label="Code"
          name="code"
          maxLength={40}
          value={values.code}
          onChange={(code) => set('code', code)}
          error={problemOf(problems, 'code')}
        />
      </div>
    </FormPanel>
  );
}

function sameValues(a: ItemValues, b: ItemValues): boolean {
  return (
    a.year === b.year && a.budgetName === b.budgetName && a.name === b.name && a.code === b.code
  );
}

/** The fields people changed, with the version (D-203); null when nothing changed. */
function changedFields(
  values: ItemValues,
  start: ItemValues,
  version: number,
): FinanceBudgetItemUpdate | null {
  const body: FinanceBudgetItemUpdate = { version };
  if (values.budgetName !== start.budgetName) {
    body.budgetName = values.budgetName;
  }
  if (values.name !== start.name) {
    body.name = values.name;
  }
  if (values.code !== start.code) {
    body.code = values.code;
  }
  return Object.keys(body).length > 1 ? body : null;
}
