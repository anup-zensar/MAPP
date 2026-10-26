import { Link } from 'react-router';
import { businessUnitOf, isEmptyValue, isVisible } from './rules';
import type { AttributeForm, AttributeValues } from './rules';
import styles from './ShellAttributeFields.module.css';
import {
  conditionUnits,
  hasConditions,
  NO_BUSINESS_UNIT_NOTE,
  shownBecauseText,
  valueText,
} from './texts';

export interface ShellAttributeValuesProps {
  form: AttributeForm;
  values: AttributeValues;
  /** The plan's Business unit option ids (records below the plan); null for a plan itself. */
  businessUnit: string[] | null;
  /** The plan's address, linked from the "appear once the business unit is chosen" note. */
  planLink?: string;
  /** The heading of each attribute section; 'none' when the details have their own heading. */
  sectionHeading?: 'h2' | 'h3' | 'none';
}

/**
 * A record's Shell attribute values, read-only (feature 002, FR-024 – FR-027): the shown
 * attributes by section, "—" for empty ones, Yes or No for yes/no, "(inactive)" for inactive
 * attributes and options; hidden attributes are left out, with the same notes as the form.
 */
export function ShellAttributeValues({
  form,
  values,
  businessUnit,
  planLink,
  sectionHeading = 'h3',
}: ShellAttributeValuesProps) {
  const units = businessUnitOf(form, values, businessUnit);
  const Heading = sectionHeading === 'none' ? null : sectionHeading;

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
            <dl className={styles.values}>
              {shown.map((attribute) => (
                <div key={attribute.id} className={styles.pair}>
                  <dt>{attribute.label}</dt>
                  <dd>
                    {valueText(attribute, values[attribute.id])}
                    {!attribute.active && ' (inactive)'}
                  </dd>
                </div>
              ))}
            </dl>
          </div>
        );
      })}
    </div>
  );
}
