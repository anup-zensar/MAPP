import { ChoiceField } from '@/ui/ChoiceField/ChoiceField';
import { cx } from '@/ui/cx';
import { FieldShell } from '@/ui/FieldShell/FieldShell';
import inputs from '@/ui/FieldShell/inputs.module.css';
import type { BulkField } from './bulk-types';

export interface ValueDraft {
  date: string;
  optionIds: string[];
  yesNo: '' | 'yes' | 'no';
}

export interface FieldValueInputProps {
  field: BulkField;
  draft: ValueDraft;
  onChange: (draft: ValueDraft) => void;
}

const YES_NO = [
  { id: 'yes', label: 'Yes', active: true },
  { id: 'no', label: 'No', active: true },
];

/** The value control of Change field, by the field's kind (FR-009). */
export function FieldValueInput({ field, draft, onChange }: FieldValueInputProps) {
  const options = field.options.map((o) => ({ id: o.id, label: o.label, active: true }));
  switch (field.kind) {
    case 'date':
      return (
        <FieldShell id="value" label="Value">
          {(describedBy) => (
            <input
              id="value"
              type="date"
              value={draft.date}
              aria-describedby={describedBy}
              className={cx(inputs.input)}
              onChange={(event) => onChange({ ...draft, date: event.target.value })}
            />
          )}
        </FieldShell>
      );
    case 'yesNo':
      return (
        <ChoiceField
          label="Value"
          name="value"
          options={YES_NO}
          value={draft.yesNo}
          onChange={(next) => onChange({ ...draft, yesNo: next === 'yes' ? 'yes' : 'no' })}
        />
      );
    case 'multipleChoice':
      return (
        <ChoiceField
          multiple
          label="Value"
          name="value"
          options={options}
          value={draft.optionIds}
          onChange={(next) => onChange({ ...draft, optionIds: next })}
        />
      );
    case 'singleChoice':
      return (
        <ChoiceField
          label="Value"
          name="value"
          options={options}
          value={draft.optionIds[0] ?? ''}
          onChange={(next) => onChange({ ...draft, optionIds: next ? [next] : [] })}
        />
      );
  }
}
