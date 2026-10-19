/**
 * The rules of Shell attribute values on forms (feature 002, data-model §4.3 and §9, FR-024 –
 * FR-027), mirrored from Go's settings form.go for instant feedback. The API stays the judge:
 * saving always goes through CheckAttributes on the server, with the same cases in form_test.go.
 */
import type { components } from '@/api/schema';
import { formatHundredths, toHundredths } from '@/lib/decimal';

export type AttributeForm = components['schemas']['AttributeForm'];
export type FormAttribute = components['schemas']['FormAttribute'];
export type FormOption = components['schemas']['FormOption'];
export type AttributeValues = components['schemas']['AttributeValues'];
export type AttributeValue = components['schemas']['AttributeValue'];
export type ClearedValue = components['schemas']['ClearedValue'];
type MissingField = components['schemas']['MissingField'];
type SplitShare = components['schemas']['SplitShare'];

/** The field name of an attribute in requests, problems and input ids: "attributes.SA-1003". */
export function attributeField(attributeId: string): string {
  return `attributes.${attributeId}`;
}

/** Every attribute of the form, in display order. */
export function allAttributes(form: AttributeForm): FormAttribute[] {
  return form.sections.flatMap((section) => section.attributes);
}

/**
 * The Business unit option ids that decide which attributes are shown: the plan's (for the forms
 * below the plan, given as `businessUnit`), or on the plan form (null) its own Business unit value.
 */
export function businessUnitOf(
  form: AttributeForm,
  values: AttributeValues,
  businessUnit: string[] | null,
): string[] {
  if (businessUnit) {
    return businessUnit;
  }
  return form.businessUnitAttributeId ? optionIdsOf(values[form.businessUnitAttributeId]) : [];
}

/** An attribute is shown when it has no condition or the business unit includes one of its options. */
export function isVisible(attribute: FormAttribute, businessUnit: string[]): boolean {
  return (
    attribute.conditionOptionIds.length === 0 ||
    attribute.conditionOptionIds.some((id) => businessUnit.includes(id))
  );
}

/** The attributes shown for this business unit, in display order (hidden ones keep their values). */
export function visibleAttributes(form: AttributeForm, businessUnit: string[]): FormAttribute[] {
  return allAttributes(form).filter((attribute) => isVisible(attribute, businessUnit));
}

/** The option ids a value holds: one, several, or those of a split; none for other types. */
export function optionIdsOf(value: AttributeValue | undefined): string[] {
  if (typeof value === 'string') {
    return value ? [value] : [];
  }
  if (Array.isArray(value)) {
    return value.map((item) => (typeof item === 'string' ? item : item.optionId));
  }
  return [];
}

/** Whether an attribute has no value (a missing yes/no reads as No, which is a value). */
export function isEmptyValue(attribute: FormAttribute, value: AttributeValue | undefined): boolean {
  if (value === undefined || value === null) {
    return attribute.type !== 'yesNo';
  }
  if (typeof value === 'string') {
    return value.trim() === '';
  }
  if (Array.isArray(value)) {
    return value.length === 0;
  }
  return false;
}

/**
 * The options that can be chosen now: active ones, and for a dependent attribute only those linked
 * to a selected option of its parent (FR-025). Options already chosen are not included unless they
 * qualify; forms add them so they stay visible.
 */
export function allowedOptions(attribute: FormAttribute, values: AttributeValues): FormOption[] {
  const active = attribute.options.filter((option) => option.active);
  if (!attribute.parentAttributeId) {
    return active;
  }
  const parentChoice = optionIdsOf(values[attribute.parentAttributeId]);
  return active.filter((option) => option.parentOptionIds.some((id) => parentChoice.includes(id)));
}

/** The attribute a dependent attribute depends on, when it is on the form. */
export function parentOf(attribute: FormAttribute, form: AttributeForm): FormAttribute | undefined {
  return attribute.parentAttributeId
    ? allAttributes(form).find((candidate) => candidate.id === attribute.parentAttributeId)
    : undefined;
}

/**
 * The parent has a value, but none of its chosen options has an active linked option (D-034):
 * the form says "No options set up for this <parent> — ask an administrator." and the attribute
 * never counts as missing.
 */
export function noOptionsForParent(attribute: FormAttribute, values: AttributeValues): boolean {
  return (
    attribute.parentAttributeId !== null &&
    optionIdsOf(values[attribute.parentAttributeId]).length > 0 &&
    allowedOptions(attribute, values).length === 0
  );
}

/**
 * The dependent values that saving would clear (FR-025): values whose parent changed since
 * `previous` (the stored record; {} on a create form) so that they are no longer linked to a chosen
 * parent option. Chains are followed in display order, so clearing a parent's value clears its
 * dependents too. The form lists these in ClearValuesDialog before saving.
 */
export function valuesToClear(
  form: AttributeForm,
  values: AttributeValues,
  previous: AttributeValues = {},
): ClearedValue[] {
  const current: AttributeValues = { ...values };
  const cleared: ClearedValue[] = [];
  for (const attribute of allAttributes(form)) {
    const parentId = attribute.parentAttributeId;
    if (!parentId || sameOptions(current[parentId], previous[parentId])) {
      continue;
    }
    const parentChoice = optionIdsOf(current[parentId]);
    const misfits = attribute.options.filter(
      (option) =>
        optionIdsOf(current[attribute.id]).includes(option.id) &&
        !option.parentOptionIds.some((id) => parentChoice.includes(id)),
    );
    if (misfits.length > 0) {
      cleared.push({
        attributeId: attribute.id,
        label: attribute.label,
        optionIds: misfits.map((option) => option.id),
        optionLabels: misfits.map((option) => option.label),
      });
      current[attribute.id] = withoutOptions(current[attribute.id], misfits);
    }
  }
  return cleared;
}

/**
 * The needed attributes that are still empty (FR-026, FR-027), as the API's MissingField: active,
 * shown, not yes/no, with a marking other than optional, and empty — or a split whose shares do
 * not add up to 100.00 %. Never missing: hidden or inactive attributes, and dependents whose chosen
 * parent options have no options (D-034). A value that is an inactive option still counts.
 */
export function missingAttributes(
  form: AttributeForm,
  values: AttributeValues,
  businessUnit: string[],
): MissingField[] {
  const missing: MissingField[] = [];
  for (const attribute of visibleAttributes(form, businessUnit)) {
    if (
      !attribute.active ||
      attribute.marking === 'optional' ||
      attribute.type === 'yesNo' ||
      noOptionsForParent(attribute, values)
    ) {
      continue;
    }
    const value = values[attribute.id];
    const field = attributeField(attribute.id);
    if (isEmptyValue(attribute, value)) {
      missing.push({ field, label: attribute.label });
    } else if (attribute.type === 'multipleChoiceSplit' && Array.isArray(value)) {
      const total = (value as SplitShare[]).reduce(
        (sum, part) => sum + (toHundredths(part.share) ?? 0n),
        0n,
      );
      if (total !== 10_000n) {
        missing.push({
          field,
          label: attribute.label,
          reason: `Shares add up to ${formatHundredths(total)} % — they must add up to 100.00 %.`,
        });
      }
    }
  }
  return missing;
}

/** A value an auto-fill chose: the attribute and its only fitting option. */
export interface AutoFilled {
  attributeId: string;
  optionId: string;
}

/**
 * After `changedAttributeId` changed: every empty dependent (and its dependents in turn) for which
 * exactly one option fits gets that option (design system §7: "Region set to 'Retail – China', the
 * only region for this cluster").
 */
export function autoFill(
  form: AttributeForm,
  values: AttributeValues,
  changedAttributeId: string,
): { values: AttributeValues; filled: AutoFilled[] } {
  let current = values;
  const filled: AutoFilled[] = [];
  const queue = [changedAttributeId];
  while (queue.length > 0) {
    const parentId = queue.shift() as string;
    for (const attribute of allAttributes(form)) {
      if (attribute.parentAttributeId !== parentId || !attribute.active) {
        continue;
      }
      const fitting = allowedOptions(attribute, current);
      if (isEmptyValue(attribute, current[attribute.id]) && fitting.length === 1) {
        const optionId = fitting[0].id;
        current = { ...current, [attribute.id]: valueOfOne(attribute, optionId) };
        filled.push({ attributeId: attribute.id, optionId });
        queue.push(attribute.id);
      }
    }
  }
  return { values: current, filled };
}

/** The value that holds exactly one option, in the attribute's own shape. */
function valueOfOne(attribute: FormAttribute, optionId: string): AttributeValue {
  switch (attribute.type) {
    case 'multipleChoice':
      return [optionId];
    case 'multipleChoiceSplit':
      return [{ optionId, share: '100.00' }];
    default:
      return optionId;
  }
}

function withoutOptions(value: AttributeValue | undefined, options: FormOption[]): AttributeValue {
  const ids = new Set(options.map((option) => option.id));
  if (typeof value === 'string') {
    return ids.has(value) ? null : value;
  }
  if (Array.isArray(value)) {
    const kept = (value as (string | SplitShare)[]).filter(
      (item) => !ids.has(typeof item === 'string' ? item : item.optionId),
    );
    return kept.length === 0 ? null : (kept as AttributeValue);
  }
  return value ?? null;
}

function sameOptions(a: AttributeValue | undefined, b: AttributeValue | undefined): boolean {
  const left = [...optionIdsOf(a)].sort();
  const right = [...optionIdsOf(b)].sort();
  return left.length === right.length && left.every((id, index) => id === right[index]);
}
