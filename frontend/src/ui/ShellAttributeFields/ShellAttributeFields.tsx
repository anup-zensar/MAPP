import { useState } from 'react';
import { Link } from 'react-router';
import {
  allAttributes,
  attributeField,
  autoFill,
  businessUnitOf,
  isEmptyValue,
  isVisible,
  parentOf,
  valuesToClear,
} from './rules';
import type { AttributeForm, AttributeValue, AttributeValues, FormAttribute } from './rules';
import { AttributeField } from './AttributeField';
import styles from './ShellAttributeFields.module.css';
import {
  autoFilledText,
  conditionUnits,
  hasConditions,
  NO_BUSINESS_UNIT_NOTE,
  shownBecauseText,
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
  /** An attribute to draw on the hover grey (the Settings preview). */
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
