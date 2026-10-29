import type { components } from '@/api/generated/schema';
import { reply } from '../replies';

type FieldHelpEntry = components['schemas']['FieldHelpEntry'];
type FieldHelpLevel = components['schemas']['FieldHelpLevel'];

const LEVELS: FieldHelpLevel[] = [
  'plan',
  'campaign',
  'initiative',
  'activity',
  'lineItem',
  'phasing',
  'common',
  'allWork',
];

function entry(key: string, text: string): FieldHelpEntry {
  const prefix = key.split('.')[0];
  return {
    key,
    level: LEVELS.find((level) => level === prefix) ?? 'common',
    label: key,
    text,
    defaultText: text,
    isDefault: text !== '',
    version: 0,
    updatedAt: null,
    updatedBy: null,
  };
}

/** GET /api/v1/field-help answering `texts` by key; an empty text is an emptied one (no ⓘ). */
export function fieldHelpReply(texts: Record<string, string>) {
  const items = Object.entries(texts).map(([key, text]) => entry(key, text));
  return reply('GET /api/v1/field-help', {
    body: { items, total: items.length, shellAttributesWithoutHelp: 0 },
  });
}

/** Help texts for every key in `keys`: "Help of <key>." */
export function helpOf(keys: string[]): Record<string, string> {
  return Object.fromEntries(keys.map((key) => [key, `Help of ${key}.`]));
}
