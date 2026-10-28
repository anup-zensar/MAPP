import { Plus } from 'lucide-react';
import { useId, useState } from 'react';
import { useFieldArray, useWatch } from 'react-hook-form';
import type { Control } from 'react-hook-form';
import { Button } from '@/ui/Button/Button';
import type { ChoiceOption } from '@/ui/ChoiceField/ChoiceField';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import type { AttributeForm } from '@/ui/ShellAttributeFields/rules';
import type { PurchaseRequest } from './api';
import { problemAt } from './form-problems';
import type { PurchaseRequestFormValues } from './form-schema';
import { lineFromOption } from './form-state';
import { ADD_LINE, NOT_SAVED_LINE } from './form-text';
import { LineItemPicker } from './LineItemPicker';
import type { LineItemPickerProps } from './LineItemPicker';
import { LineRow } from './LineRow';
import { RemoveLineButton } from './RemoveLineDialog';
import styles from './LineRowsEditor.module.css';

export interface LineRowsEditorProps {
  control: Control<PurchaseRequestFormValues>;
  problems: FieldProblem[];
  currencies: ChoiceOption[];
  attributeForm: AttributeForm;
  /** The PR as stored; null on New purchase request. */
  stored: PurchaseRequest | null;
  optionsQuery: LineItemPickerProps['query'];
}

/** The lines of the form (032 US4): each line's fields, Add line through the picker, Remove line. */
export function LineRowsEditor({
  control,
  problems,
  currencies,
  attributeForm,
  stored,
  optionsQuery,
}: LineRowsEditorProps) {
  const reasonId = useId();
  const [picking, setPicking] = useState(false);
  const { fields, append, remove } = useFieldArray({ control, name: 'lines', keyName: 'key' });
  const lines = useWatch({ control, name: 'lines' });
  const addLine = stored?.access.addLine ?? { available: true, reason: '' };
  const linesProblem = problemAt(problems, 'lines');

  return (
    <div className={styles.editor}>
      {linesProblem && (
        <p id="lines" tabIndex={-1} className={styles.problem}>
          {linesProblem}
        </p>
      )}
      {fields.map((field, index) => {
        const line = lines[index] ?? field;
        const storedLine = stored?.lines.find((each) => each.id === line.id);
        return (
          <LineRow
            key={field.key}
            index={index}
            line={line}
            control={control}
            problems={problems}
            currencies={currencies}
            attributeForm={attributeForm}
            mark={stored && !storedLine ? NOT_SAVED_LINE : undefined}
            removal={
              <RemoveLineButton
                lineItemId={line.lineItem.id}
                reserved={storedLine?.amount ?? null}
                availability={storedLine?.access.remove ?? null}
                isLast={fields.length === 1}
                onRemove={() => remove(index)}
              />
            }
          />
        );
      })}
      {(addLine.available || addLine.reason) && (
        <div className={styles.add}>
          <Button
            icon={<Plus />}
            disabled={!addLine.available}
            aria-describedby={addLine.available ? undefined : reasonId}
            onClick={() => setPicking(true)}
          >
            {ADD_LINE}
          </Button>
          {!addLine.available && (
            <p id={reasonId} className={styles.reason}>
              {addLine.reason}
            </p>
          )}
        </div>
      )}
      <LineItemPicker
        open={picking}
        onOpenChange={setPicking}
        query={optionsQuery}
        inForm={lines.map((line) => line.lineItem.id)}
        onChoose={(option) => append(lineFromOption(option))}
      />
    </div>
  );
}
