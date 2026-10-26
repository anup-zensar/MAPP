/** The words of Settings › Field help (feature 016, US7). */
import { formatWhole } from '@/format/decimal';
import type { FieldHelpLevel } from './fieldHelpApi';

/** The groups of the page, in the order of the catalogue (data-model §6). */
export const HELP_LEVELS: FieldHelpLevel[] = [
  'plan',
  'campaign',
  'initiative',
  'activity',
  'lineItem',
  'phasing',
  'common',
  'allWork',
];

/** What each group is called. */
export const HELP_LEVEL_WORDS: Record<FieldHelpLevel, string> = {
  plan: 'Plan',
  campaign: 'Campaign',
  initiative: 'Initiative',
  activity: 'Activity',
  lineItem: 'Line item',
  phasing: 'Budget by month',
  common: 'Every record',
  allWork: 'All work',
};

export function isHelpLevel(value: string | null): value is FieldHelpLevel {
  return HELP_LEVELS.includes(value as FieldHelpLevel);
}

/** "Plan › Plan budget", as the activity log names an entry (data-model §7). */
export function entryName(level: FieldHelpLevel, label: string): string {
  return `${HELP_LEVEL_WORDS[level]} › ${label}`;
}

/** The text is longer than the API takes (FR-046). */
export const TOO_LONG = 'Use 300 characters or fewer.';

/** "3 Shell attributes have no help yet" (FR-048). */
export function attributesWithoutHelp(count: number): string {
  return count === 1
    ? '1 Shell attribute has no help yet'
    : `${formatWhole(count)} Shell attributes have no help yet`;
}

/** Where the Shell attributes without help are listed (FR-048). */
export const MISSING_HELP_PATH = '/settings/shell-attributes?help=missing';
