import type { components } from '@/api/generated/schema';
import { DEFAULT_SECTION, isChoiceType } from './attribute-words';
import type { AttributeLevel, FormAttribute } from './attribute-words';
import type { ShellAttribute, ShellAttributeCreate, ShellAttributeUpdate } from './api';
import type { AttributeValues } from './schemas';

type FormOption = components['schemas']['FormOption'];

/** The text fields an edit can change and send (D-203: only the changed ones). */
const TEXT_FIELDS = [
  'label',
  'helpText',
  'level',
  'type',
  'marking',
  'section',
  'parentAttributeId',
] as const;

export type ChangedField = (typeof TEXT_FIELDS)[number] | 'conditionOptionIds';

export function startValues(
  attribute: ShellAttribute | undefined,
  level: AttributeLevel,
): AttributeValues {
  return {
    label: attribute?.label ?? '',
    helpText: attribute?.helpText ?? '',
    level,
    type: attribute?.type ?? '',
    marking: attribute?.marking ?? 'optional',
    section: attribute?.section ?? DEFAULT_SECTION,
    parentAttributeId: attribute?.parentAttribute?.id ?? '',
    conditionOptionIds: attribute?.condition.map((option) => option.id) ?? [],
    options: [],
    optionLinks: {},
  };
}

export function changedFields(values: AttributeValues, start: AttributeValues): ChangedField[] {
  return [
    ...TEXT_FIELDS.filter((field) => values[field] !== start[field]),
    ...(sameIds(values.conditionOptionIds, start.conditionOptionIds)
      ? []
      : (['conditionOptionIds'] as const)),
  ];
}

/**
 * The changed fields, the links a new parent needs (`optionsToLink`, null when none are needed)
 * and the version the person started from.
 */
export function updateBody(
  values: AttributeValues,
  changed: ChangedField[],
  version: number,
  optionsToLink: FormOption[] | null,
): ShellAttributeUpdate {
  const body: ShellAttributeUpdate = { version };
  for (const field of changed) {
    if (field === 'conditionOptionIds') {
      body.conditionOptionIds = values.conditionOptionIds;
    } else {
      body[field] = values[field];
    }
  }
  if (optionsToLink) {
    body.optionLinks = optionsToLink.map((option) => ({
      optionId: option.id,
      parentOptionIds: values.optionLinks[option.id] ?? [],
    }));
  }
  return body;
}

export function createBody(
  values: AttributeValues,
  parent: FormAttribute | undefined,
): ShellAttributeCreate {
  const isChoice = isChoiceType(values.type);
  return {
    label: values.label,
    helpText: values.helpText,
    level: values.level,
    type: values.type,
    marking: values.marking,
    section: values.section,
    parentAttributeId: isChoice ? values.parentAttributeId : '',
    conditionOptionIds: values.conditionOptionIds,
    options: isChoice
      ? values.options.map((row) =>
          parent
            ? { label: row.label, parentOptionIds: row.parentOptionIds }
            : { label: row.label },
        )
      : [],
  };
}

function sameIds(a: string[], b: string[]): boolean {
  return a.length === b.length && a.every((id) => b.includes(id));
}
