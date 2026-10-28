// Shell attributes' words and the FR-017 rules of what the editor offers; the API stays the judge.
import type { components } from '@/api/generated/schema';
import { midSentence } from '@/format/text';

export type AttributeLevel = components['schemas']['AttributeLevel'];
export type AttributeType = components['schemas']['AttributeType'];
export type AttributeMarking = components['schemas']['AttributeMarking'];

export const LEVELS = [
  'plan',
  'campaign',
  'initiative',
  'activity',
  'lineItem',
  'purchaseRequest',
  'purchaseRequestLine',
] as const satisfies readonly AttributeLevel[];

export const LEVEL_WORDS: Record<AttributeLevel, string> = {
  plan: 'Plan',
  campaign: 'Campaign',
  initiative: 'Initiative',
  activity: 'Activity',
  lineItem: 'Line item',
  purchaseRequest: 'Purchase request',
  purchaseRequestLine: 'PR line',
};

export const TYPE_WORDS: Record<AttributeType, string> = {
  singleChoice: 'Single choice',
  multipleChoice: 'Multiple choice',
  multipleChoiceSplit: 'Multiple choice with % split',
  yesNo: 'Yes/no',
  text: 'Text',
  longText: 'Long text',
  number: 'Number',
};

export const MARKING_WORDS: Record<AttributeMarking, string> = {
  optional: 'Optional',
  neededToPublish: 'Needed to publish',
  neededToSubmit: 'Needed to submit',
};

const TYPES: AttributeType[] = [
  'singleChoice',
  'multipleChoice',
  'multipleChoiceSplit',
  'yesNo',
  'text',
  'longText',
  'number',
];

function carriesCost(level: AttributeLevel): boolean {
  return level === 'activity' || level === 'lineItem';
}

export function levelMidSentence(level: AttributeLevel): string {
  return LEVEL_WORDS[level].toLowerCase();
}

export function isLevel(value: string | null): value is AttributeLevel {
  return LEVELS.some((level) => level === value);
}

export function isChoiceType(type: string): boolean {
  return type === 'singleChoice' || type === 'multipleChoice' || type === 'multipleChoiceSplit';
}

export function typesFor(level: AttributeLevel): AttributeType[] {
  return TYPES.filter((type) => type !== 'multipleChoiceSplit' || carriesCost(level));
}

/** Needed to submit where records are submitted for approval (FR-017, 032 FR-006), else to publish. */
export function markingsFor(level: AttributeLevel): AttributeMarking[] {
  const submitted =
    carriesCost(level) || level === 'purchaseRequest' || level === 'purchaseRequestLine';
  return ['optional', submitted ? 'neededToSubmit' : 'neededToPublish'];
}

type ShellAttribute = components['schemas']['ShellAttribute'];

const OR_LIST = new Intl.ListFormat('en-GB', { type: 'disjunction' });

export function dependsOnText(attribute: ShellAttribute): string | null {
  return attribute.parentAttribute ? `Depends on ${attribute.parentAttribute.name}` : null;
}

export function conditionText(attribute: ShellAttribute): string | null {
  const names = attribute.condition.map((option) => option.name);
  return names.length > 0 ? OR_LIST.format(names) : null;
}

/** When the attribute is shown: its business-unit condition, or "Always shown" (FR-023). */
export function shownWhenText(attribute: ShellAttribute): string {
  const condition = conditionText(attribute);
  return condition ? `Shown when the business unit includes ${condition}` : 'Always shown';
}

export function lockReason(attribute: ShellAttribute, field: string): string | undefined {
  return attribute.locks.find((lock) => lock.field === field)?.reason;
}

export type FormAttribute = components['schemas']['FormAttribute'];
type AttributeForm = components['schemas']['AttributeForm'];

export function findFormAttribute(form: AttributeForm, id: string): FormAttribute | undefined {
  return form.sections
    .flatMap((section) => section.attributes)
    .find((attribute) => attribute.id === id);
}

/** The section of an attribute whose Section field is left empty (FR-019). */
export const DEFAULT_SECTION = 'Shell attributes';

export function linkOptionText(optionLabel: string, parentLabel: string): string {
  return `Link '${optionLabel.trim()}' to at least one ${midSentence(parentLabel)} option.`;
}

export function linkedToHint(parentLabel: string): string {
  return `Shown when one of these ${midSentence(parentLabel)} options is chosen.`;
}

/** Choice attributes before this one (FR-022); a new or moved one goes last in its section. */
export function parentCandidates(
  all: ShellAttribute[],
  level: AttributeLevel,
  section: string,
  attribute?: ShellAttribute,
): ShellAttribute[] {
  const target = section.trim() || DEFAULT_SECTION;
  const levelList = all.filter((each) => each.level === level);
  let before: ShellAttribute[];
  const ownPlace = attribute ? levelList.findIndex((each) => each.id === attribute.id) : -1;
  if (attribute && ownPlace !== -1 && attribute.section === target) {
    before = levelList.slice(0, ownPlace);
  } else {
    const others = levelList.filter((each) => each.id !== attribute?.id);
    const lastOfSection = others.findLastIndex((each) => each.section === target);
    before = lastOfSection === -1 ? others : others.slice(0, lastOfSection + 1);
  }
  return before.filter((each) => each.type === 'singleChoice' || each.type === 'multipleChoice');
}
