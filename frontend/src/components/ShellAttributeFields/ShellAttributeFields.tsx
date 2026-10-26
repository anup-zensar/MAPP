import { useState } from 'react';
import { Link } from 'react-router';
import { cx } from '@/lib/cx';
import { parseWholeInput } from '@/lib/decimal';
import { CheckboxField } from '../CheckboxField/CheckboxField';
import { ChoiceField } from '../ChoiceField/ChoiceField';
import type { ChoiceOption } from '../ChoiceField/ChoiceField';
import { DecimalField } from '../DecimalField/DecimalField';
import { SplitEditor } from '../SplitEditor/SplitEditor';
import type { SplitShare } from '../SplitEditor/SplitEditor';
import { TextAreaField } from '../TextAreaField/TextAreaField';
import { TextField } from '../TextField/TextField';
import {
  allAttributes,
  allowedOptions,
  attributeField,
  autoFill,
  businessUnitOf,
  isEmptyValue,
  isVisible,
  noOptionsForParent,
  optionIdsOf,
  parentOf,
  valuesToClear,
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
  autoFilledText,
  chooseParentFirst,
  conditionUnits,
  hasConditions,
  INACTIVE_OPTIONS_NOTE,
  inactiveAttributeText,
  markingWord,
  NO_BUSINESS_UNIT_NOTE,
  noOptionsForParentText,
  shownBecauseText,
  valueText,
  willBeClearedText,
} from './texts';

export interface ShellAttributeFieldsProps {
  /** The level's form definition: GET /api/v1/shell-attribute-forms/{level} (useAttributeForm). */
  form: AttributeForm;
  /** The record's values, keyed by attribute id (its `attributes` field). */
  values: AttributeValues;
  onChange: (values: AttributeValues) => void;
  /**
   * The plan's Business unit option ids, which decide the conditional attributes (forms below the
   * plan); null on the plan form, where its own Business unit value decides.
   */
  businessUnit: string[] | null;
  /** The values as stored (edit forms); dependents whose parent changed since are cleared on save. */
  previousValues?: AttributeValues;
  /** Problems by field name, as the API names them: "attributes.SA-1003". */
  errors?: Record<string, string>;
  /** An attribute to draw with a blue tint (the Settings preview). */
  highlightAttributeId?: string;
  /** The plan's address, linked from the "appear once the business unit is chosen" note. */
  planLink?: string;
  /** The heading of each attribute section; 'none' when the form has its own heading. */
  sectionHeading?: 'h2' | 'h3' | 'none';
}

/**
 * A level's Shell attributes on a form (feature 002, FR-024 – FR-027): grouped in their sections
 * with marking words and ⓘ help; the control for each type; dependent fields locked until their
 * parent is chosen, filtered by it and filled in when only one option fits; attributes shown only
 * for some business units, with the reason; inactive attributes and options kept read-only.
 * Hidden attributes keep their values. Input ids are "attributes.<attributeId>".
 *
 * Before saving, list `valuesToClear(form, values, previousValues)` in ClearValuesDialog.
 */
export function ShellAttributeFields({
  form,
  values,
  onChange,
  businessUnit,
  previousValues = {},
  errors = {},
  highlightAttributeId,
  planLink,
  sectionHeading = 'h3',
}: ShellAttributeFieldsProps) {
  // The auto-fill messages of the last change, by attribute id ("Region set to …").
  const [autoFilled, setAutoFilled] = useState<Record<string, string>>({});
  const units = businessUnitOf(form, values, businessUnit);
  const cleared = valuesToClear(form, values, previousValues);
  const Heading = sectionHeading === 'none' ? null : sectionHeading;

  function change(attribute: FormAttribute, value: AttributeValue | undefined) {
    const next: AttributeValues = { ...values };
    if (value === undefined || isEmptyValue(attribute, value)) {
      delete next[attribute.id];
    } else {
      next[attribute.id] = value;
    }
    const result = autoFill(form, next, attribute.id);
    const messages: Record<string, string> = {};
    for (const filled of result.filled) {
      const target = allAttributes(form).find((candidate) => candidate.id === filled.attributeId);
      const parent = target && parentOf(target, form);
      const option = target?.options.find((candidate) => candidate.id === filled.optionId);
      if (target && parent && option) {
        messages[target.id] = autoFilledText(target, option.label, parent);
      }
    }
    setAutoFilled(messages);
    onChange(result.values);
  }

  return (
    <div className={styles.fields}>
      {units.length === 0 && hasConditions(form) && (
        <p className={styles.note}>
          {NO_BUSINESS_UNIT_NOTE}
          {planLink && (
            <>
              {' '}
              <Link to={planLink}>Open the plan</Link>
            </>
          )}
        </p>
      )}
      {form.sections.map((section) => {
        const shown = section.attributes.filter(
          (attribute) =>
            isVisible(attribute, units) &&
            (attribute.active || !isEmptyValue(attribute, values[attribute.id])),
        );
        if (shown.length === 0) {
          return null;
        }
        return (
          <div key={section.name} className={styles.section}>
            {Heading && <Heading className={styles.heading}>{section.name}</Heading>}
            {section.conditional && (
              <p className={styles.note}>{shownBecauseText(conditionUnits(form, shown, units))}</p>
            )}
            {shown.map((attribute) => (
              <AttributeField
                key={attribute.id}
                form={form}
                attribute={attribute}
                values={values}
                error={errors[attributeField(attribute.id)]}
                cleared={cleared.find((item) => item.attributeId === attribute.id)}
                autoFilled={autoFilled[attribute.id]}
                conditionNote={
                  !section.conditional && attribute.conditionOptionIds.length > 0
                    ? shownBecauseText(conditionUnits(form, [attribute], units))
                    : undefined
                }
                highlighted={attribute.id === highlightAttributeId}
                onChange={(value) => change(attribute, value)}
              />
            ))}
          </div>
        );
      })}
    </div>
  );
}

interface AttributeFieldProps {
  form: AttributeForm;
  attribute: FormAttribute;
  values: AttributeValues;
  error?: string;
  cleared?: ClearedValue;
  autoFilled?: string;
  conditionNote?: string;
  highlighted: boolean;
  onChange: (value: AttributeValue | undefined) => void;
}

/** One attribute: its control with its notes underneath. */
function AttributeField({
  form,
  attribute,
  values,
  error,
  cleared,
  autoFilled,
  conditionNote,
  highlighted,
  onChange,
}: AttributeFieldProps) {
  const name = attributeField(attribute.id);
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
              value={Array.isArray(value) ? (value as SplitShare[]) : []}
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
