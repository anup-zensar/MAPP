/**
 * The words of Shell attributes on screen (feature 002 plan, "User-visible messages": Attribute
 * list, Attribute words; data model §4) and the rules of FR-017 that decide what the editor
 * offers. The API stays the judge: these only shape what is offered.
 */
import type { components } from '@/api/generated/schema';
import { midSentence } from '@/format/text';

export type AttributeLevel = components['schemas']['AttributeLevel'];
export type AttributeType = components['schemas']['AttributeType'];
export type AttributeMarking = components['schemas']['AttributeMarking'];

/** The levels in the order of the planning hierarchy. */
export const LEVELS = [
  'plan',
  'campaign',
  'initiative',
  'activity',
  'lineItem',
] as const satisfies readonly AttributeLevel[];

export const LEVEL_WORDS: Record<AttributeLevel, string> = {
  plan: 'Plan',
  campaign: 'Campaign',
  initiative: 'Initiative',
  activity: 'Activity',
  lineItem: 'Line item',
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

/** The types in the order the editor offers them. */
const TYPES: AttributeType[] = [
  'singleChoice',
  'multipleChoice',
  'multipleChoiceSplit',
  'yesNo',
  'text',
  'longText',
  'number',
];

/** True for a level whose records carry cost: activities and line items (FR-017). */
function carriesCost(level: AttributeLevel): boolean {
  return level === 'activity' || level === 'lineItem';
}

/** A level as it reads mid-sentence: "No line item attributes yet". */
export function levelMidSentence(level: AttributeLevel): string {
  return LEVEL_WORDS[level].toLowerCase();
}

/** True when the value is one of the levels (a level from the address may be anything). */
export function isLevel(value: string | null): value is AttributeLevel {
  return LEVELS.some((level) => level === value);
}

/** Choice types have options, at least one (FR-018). */
export function isChoiceType(type: string): boolean {
  return type === 'singleChoice' || type === 'multipleChoice' || type === 'multipleChoiceSplit';
}

/** The types a level offers: "Multiple choice with % split" only where records carry cost. */
export function typesFor(level: AttributeLevel): AttributeType[] {
  return TYPES.filter((type) => type !== 'multipleChoiceSplit' || carriesCost(level));
}

/** Optional and Needed to publish above activities; Optional and Needed to submit below (FR-017). */
export function markingsFor(level: AttributeLevel): AttributeMarking[] {
  return ['optional', carriesCost(level) ? 'neededToSubmit' : 'neededToPublish'];
}

type ShellAttribute = components['schemas']['ShellAttribute'];

/** "Lubricants or Mobility": the condition holds when the business unit includes any of them. */
const OR_LIST = new Intl.ListFormat('en-GB', { type: 'disjunction' });

/** "Depends on Cluster", or null when its options are not filtered by another attribute. */
export function dependsOnText(attribute: ShellAttribute): string | null {
  return attribute.parentAttribute ? `Depends on ${attribute.parentAttribute.name}` : null;
}

/** The business units of the condition, "Lubricants or Mobility"; null when always shown. */
export function conditionText(attribute: ShellAttribute): string | null {
  const names = attribute.condition.map((option) => option.name);
  return names.length > 0 ? OR_LIST.format(names) : null;
}

/** When the attribute is shown: its business-unit condition, or "Always shown" (FR-023). */
export function shownWhenText(attribute: ShellAttribute): string {
  const condition = conditionText(attribute);
  return condition ? `Shown when the business unit includes ${condition}` : 'Always shown';
}

/** The reason a field or action of the attribute is locked (data model §4.1), if it is. */
export function lockReason(attribute: ShellAttribute, field: string): string | undefined {
  return attribute.locks.find((lock) => lock.field === field)?.reason;
}

export type FormAttribute = components['schemas']['FormAttribute'];
type AttributeForm = components['schemas']['AttributeForm'];

/** One attribute of a level's form (GET /api/v1/shell-attribute-forms/{level}), with its options. */
export function findFormAttribute(form: AttributeForm, id: string): FormAttribute | undefined {
  return form.sections
    .flatMap((section) => section.attributes)
    .find((attribute) => attribute.id === id);
}

/** The section of an attribute whose Section field is left empty (FR-019). */
export const DEFAULT_SECTION = 'Shell attributes';

/** "Link 'RETAIL – CHINA' to at least one cluster option." (FR-022, as the API says it.) */
export function linkOptionText(optionLabel: string, parentLabel: string): string {
  return `Link '${optionLabel.trim()}' to at least one ${midSentence(parentLabel)} option.`;
}

/** "Shown when one of these cluster options is chosen." — the hint of Linked to. */
export function linkedToHint(parentLabel: string): string {
  return `Shown when one of these ${midSentence(parentLabel)} options is chosen.`;
}

/**
 * The attributes an attribute may depend on (FR-022, data model §4.2): the single and multiple
 * choice attributes of `level` placed before it. An attribute that keeps its section keeps its
 * place; a new attribute, or one moved to another section, goes last in that section (or last on
 * the form when the section is new). `level` lists every attribute in display order. The API
 * checks again when saving.
 */
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
