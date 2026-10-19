/**
 * The texts of Shell attributes on forms and details (feature 002 plan, "User-visible messages",
 * rows "Attribute fields on forms" and "Split editor"), and how a value reads.
 */
import type { components } from '@/api/schema';
import { formatPercent, formatWhole } from '@/lib/decimal';
import { joinWithAnd, midSentence } from '@/lib/text';
import { allAttributes, isEmptyValue } from './rules';
import type { AttributeForm, AttributeValue, FormAttribute } from './rules';

type SplitShare = components['schemas']['SplitShare'];

/** Shown for an empty value (design system §6.3). */
export const EMPTY_VALUE = '—';

export const NO_BUSINESS_UNIT_NOTE =
  "More Shell attributes appear once the plan's business unit is chosen.";
export const INACTIVE_OPTIONS_NOTE =
  "Inactive options stay on this record but can't be chosen again.";

export function markingWord(
  marking: FormAttribute['marking'],
): 'Needed to publish' | 'Needed to submit' | undefined {
  switch (marking) {
    case 'neededToPublish':
      return 'Needed to publish';
    case 'neededToSubmit':
      return 'Needed to submit';
    default:
      return undefined;
  }
}

/** "Choose a cluster first." */
export function chooseParentFirst(parent: FormAttribute): string {
  return `Choose a ${midSentence(parent.label)} first.`;
}

/** "No options set up for this cluster — ask an administrator." (D-034) */
export function noOptionsForParentText(parent: FormAttribute): string {
  return `No options set up for this ${midSentence(parent.label)} — ask an administrator.`;
}

/** "Region set to 'RETAIL – CHINA', the only region for this cluster." */
export function autoFilledText(attribute: FormAttribute, option: string, parent: FormAttribute) {
  return `${attribute.label} set to '${option}', the only ${midSentence(attribute.label)} for this ${midSentence(parent.label)}.`;
}

/** "Region is inactive — its value is kept and can't be changed." */
export function inactiveAttributeText(attribute: FormAttribute): string {
  return `${attribute.label} is inactive — its value is kept and can't be changed.`;
}

/** "GL code '6100' will be cleared — it doesn't fit 'Media'." */
export function willBeClearedText(
  attribute: FormAttribute,
  optionLabels: string[],
  parentLabels: string[],
): string {
  const options = optionLabels.map((label) => `'${label}'`).join(', ');
  return parentLabels.length > 0
    ? `${attribute.label} ${options} will be cleared — it doesn't fit ${joinWithAnd(parentLabels.map((label) => `'${label}'`))}.`
    : `${attribute.label} ${options} will be cleared.`;
}

/** "Shown because the plan's business unit includes Lubricants and Mobility." */
export function shownBecauseText(units: string[]): string {
  return `Shown because the plan's business unit includes ${joinWithAnd(units)}.`;
}

/** The labels of the Business unit options in `businessUnit` that make these attributes shown. */
export function conditionUnits(
  form: AttributeForm,
  attributes: FormAttribute[],
  businessUnit: string[],
): string[] {
  const conditions = new Set(attributes.flatMap((attribute) => attribute.conditionOptionIds));
  return form.businessUnitOptions
    .filter((option) => conditions.has(option.id) && businessUnit.includes(option.id))
    .map((option) => option.label);
}

/** Whether any attribute of the form is shown only for some business units. */
export function hasConditions(form: AttributeForm): boolean {
  return allAttributes(form).some((attribute) => attribute.conditionOptionIds.length > 0);
}

/** An option's label, with " (inactive)" for an inactive one. */
export function optionLabel(attribute: FormAttribute, optionId: string): string {
  const option = attribute.options.find((candidate) => candidate.id === optionId);
  if (!option) {
    return optionId;
  }
  return option.active ? option.label : `${option.label} (inactive)`;
}

/**
 * How a value reads in details and read-only fields: option labels ("(inactive)" ones marked),
 * "Helix 60.00 %, Rimula 40.00 %" for a split, Yes or No (missing reads as No), the text, the
 * number with grouping, or "—".
 */
export function valueText(attribute: FormAttribute, value: AttributeValue | undefined): string {
  if (attribute.type === 'yesNo') {
    return value === true ? 'Yes' : 'No';
  }
  if (isEmptyValue(attribute, value)) {
    return EMPTY_VALUE;
  }
  switch (attribute.type) {
    case 'singleChoice':
      return optionLabel(attribute, value as string);
    case 'multipleChoice':
      return (value as string[]).map((id) => optionLabel(attribute, id)).join(', ');
    case 'multipleChoiceSplit':
      return (value as SplitShare[])
        .map((part) => `${optionLabel(attribute, part.optionId)} ${formatPercent(part.share)}`)
        .join(', ');
    case 'number':
      return typeof value === 'number' ? formatWhole(value) : String(value);
    default:
      return String(value);
  }
}
