import { parseWholeInput } from '@/format/decimal';
import { cx } from '../cx';
import { CheckboxField } from '../CheckboxField/CheckboxField';
import { ChoiceField } from '../ChoiceField/ChoiceField';
import type { ChoiceOption } from '../ChoiceField/ChoiceField';
import { DecimalField } from '../DecimalField/DecimalField';
import { SplitEditor } from '../SplitEditor/SplitEditor';
import { TextAreaField } from '../TextAreaField/TextAreaField';
import { TextField } from '../TextField/TextField';
import {
  allowedOptions,
  attributeField,
  noOptionsForParent,
  optionIdsOf,
  parentOf,
  splitShares,
} from './rules';
import type {
  AttributeForm,
  AttributeValue,
  AttributeValues,
  ClearedValue,
  FormAttribute,
} from './rules';
import styles from './ShellAttributeFields.module.css';
import {
  chooseParentFirst,
  INACTIVE_OPTIONS_NOTE,
  inactiveAttributeText,
  markingWord,
  noOptionsForParentText,
  valueText,
  willBeClearedText,
} from './texts';

export interface AttributeFieldProps {
  form: AttributeForm;
  attribute: FormAttribute;
  values: AttributeValues;
  error?: string;
  cleared?: ClearedValue;
  autoFilled?: string;
  conditionNote?: string;
  highlighted: boolean;
  onChange: (value: AttributeValue | undefined) => void;
  fieldPrefix?: string;
}

/** One attribute: its control with its notes underneath. */
export function AttributeField({
  form,
  attribute,
  values,
  error,
  cleared,
  autoFilled,
  conditionNote,
  highlighted,
  onChange,
  fieldPrefix = '',
}: AttributeFieldProps) {
  const name = fieldPrefix + attributeField(attribute.id);
  const value = values[attribute.id];
  const parent = parentOf(attribute, form);
  const frame = {
    label: attribute.label,
    marking: markingWord(attribute.marking),
    tip: attribute.helpText || undefined,
    error,
  };

  const notes: string[] = [];
  if (conditionNote) {
    notes.push(conditionNote);
  }
  if (cleared && parent) {
    const parentLabels = optionIdsOf(values[parent.id]).map(
      (id) => parent.options.find((option) => option.id === id)?.label ?? id,
    );
    notes.push(willBeClearedText(attribute, cleared.optionLabels, parentLabels));
  }
  const chosen = optionIdsOf(value);
  if (chosen.some((id) => attribute.options.find((option) => option.id === id)?.active === false)) {
    notes.push(INACTIVE_OPTIONS_NOTE);
  }

  let control;
  if (!attribute.active) {
    control = (
      <div className={styles.readOnly} id={name}>
        <p className={styles.readOnlyLabel}>{attribute.label}</p>
        <p className={styles.readOnlyValue}>{valueText(attribute, value)} (inactive)</p>
        <p className={styles.note}>{inactiveAttributeText(attribute)}</p>
      </div>
    );
  } else {
    switch (attribute.type) {
      case 'singleChoice':
      case 'multipleChoice':
      case 'multipleChoiceSplit': {
        const options = choiceOptions(attribute, values);
        const disabledReason =
          parent && optionIdsOf(values[parent.id]).length === 0
            ? chooseParentFirst(parent)
            : parent && noOptionsForParent(attribute, values)
              ? noOptionsForParentText(parent)
              : undefined;
        if (attribute.type === 'multipleChoiceSplit' && !disabledReason) {
          control = (
            <SplitEditor
              {...frame}
              name={name}
              options={options}
              value={splitShares(value)}
              onChange={onChange}
            />
          );
        } else if (attribute.type === 'singleChoice') {
          control = (
            <ChoiceField
              {...frame}
              name={name}
              options={options}
              value={typeof value === 'string' ? value : ''}
              onChange={onChange}
              disabledReason={disabledReason}
            />
          );
        } else {
          control = (
            <ChoiceField
              {...frame}
              name={name}
              multiple
              options={options}
              value={chosen}
              onChange={(ids) =>
                onChange(
                  attribute.type === 'multipleChoiceSplit'
                    ? ids.map((optionId) => ({ optionId, share: '' }))
                    : ids,
                )
              }
              disabledReason={disabledReason}
            />
          );
        }
        break;
      }
      case 'yesNo':
        control = (
          <CheckboxField
            label={attribute.label}
            name={name}
            tip={frame.tip}
            error={error}
            checked={value === true}
            onChange={onChange}
          />
        );
        break;
      case 'text':
        control = (
          <TextField
            {...frame}
            name={name}
            maxLength={120}
            value={typeof value === 'string' ? value : ''}
            onChange={onChange}
          />
        );
        break;
      case 'longText':
        control = (
          <TextAreaField
            {...frame}
            name={name}
            maxLength={2000}
            value={typeof value === 'string' ? value : ''}
            onChange={onChange}
          />
        );
        break;
      case 'number':
        control = (
          <DecimalField
            {...frame}
            name={name}
            decimals={0}
            value={typeof value === 'number' || typeof value === 'string' ? String(value) : ''}
            onChange={(text) => onChange(numberValue(text))}
          />
        );
        break;
    }
  }

  return (
    <div className={cx(styles.attribute, highlighted && styles.highlighted)}>
      {control}
      {notes.map((note) => (
        <p key={note} className={styles.note}>
          {note}
        </p>
      ))}
      {/* Polite, so screen readers hear when a value was filled in for them. */}
      <p aria-live="polite" className={styles.note}>
        {autoFilled ?? ''}
      </p>
    </div>
  );
}

/**
 * The options a choice offers: those that can be chosen now, plus those already chosen (inactive
 * or no longer linked to the parent), in the attribute's order.
 */
function choiceOptions(attribute: FormAttribute, values: AttributeValues): ChoiceOption[] {
  const allowed = new Set(allowedOptions(attribute, values).map((option) => option.id));
  const chosen = optionIdsOf(values[attribute.id]);
  return attribute.options
    .filter((option) => allowed.has(option.id) || chosen.includes(option.id))
    .map((option) => ({ id: option.id, label: option.label, active: option.active }));
}

/** A typed whole number as the API takes it; text that is not one is sent as typed, for the API to name. */
function numberValue(text: string): AttributeValue | undefined {
  if (text.trim() === '') {
    return undefined;
  }
  const whole = parseWholeInput(text);
  return whole !== null &&
    whole <= BigInt(Number.MAX_SAFE_INTEGER) &&
    whole >= -BigInt(Number.MAX_SAFE_INTEGER)
    ? Number(whole)
    : text;
}
