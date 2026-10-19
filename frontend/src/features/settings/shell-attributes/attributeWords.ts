/**
 * The words of Shell attributes on screen (feature 002 plan, "User-visible messages": Attribute
 * list, Attribute words; data model §4) and the rules of FR-017 that decide what the editor
 * offers. The API stays the judge: these only shape what is offered.
 */
import type { components } from '@/api/schema';

export type AttributeLevel = components['schemas']['AttributeLevel'];
export type AttributeType = components['schemas']['AttributeType'];
export type AttributeMarking = components['schemas']['AttributeMarking'];

/** The levels in the order of the planning hierarchy. */
export const LEVELS: AttributeLevel[] = ['plan', 'campaign', 'initiative', 'activity', 'lineItem'];

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
const TYPES = Object.keys(TYPE_WORDS) as AttributeType[];

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
  return LEVELS.includes(value as AttributeLevel);
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
