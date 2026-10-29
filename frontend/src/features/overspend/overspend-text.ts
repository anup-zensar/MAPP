import { formatMoneyText } from '@/format/decimal';
import type { Money, OverspendMode } from './api';

export const OVERSPEND_LABEL = 'When planned cost goes over this budget';
export const REFUSES_OVERSPENDING = 'Refuses overspending';
export const OVERSPEND_CONTROL = 'Overspend control';
export const DEFAULT_LABEL = 'New budgets start with';
export const SETTINGS_DESCRIPTION =
  'Whether a new budget warns or refuses when the planned cost under it goes over it.';
export const EXISTING_KEEP = 'Existing budgets keep their own setting.';
export const SETTING_SAVED = 'Overspend control saved.';
export const SAVE = 'Save';

export const MODES: readonly OverspendMode[] = ['warn', 'refuse'];

export const MODE_WORDS: Record<OverspendMode, string> = { warn: 'Warn', refuse: 'Refuse' };

export const MODE_HINTS: Record<OverspendMode, string> = {
  warn: 'Save and show the overage',
  refuse: 'Refuse any save that takes the planned cost over the budget',
};

const LEVEL_WORDS: Record<string, string> = {
  plan: 'plan',
  campaign: 'campaign',
  initiative: 'initiative',
  globalBudget: 'global budget',
};

export function levelWord(type: string): string {
  return LEVEL_WORDS[type] ?? type;
}

export function budgetBelowPlannedText(planned: Money): string {
  return `The planned cost is ${formatMoneyText(planned)}; this budget refuses overspending, so it can't be lower than that.`;
}

export function alreadyOverText(over: Money): string {
  return `The planned cost is already over this budget by ${formatMoneyText(over)}. Anything that adds cost will be refused until it is back within the budget.`;
}

export function overLimitText(over: Money, type: string, name: string): string {
  return `This is ${formatMoneyText(over)} over the budget of ${levelWord(type)} '${name}', which refuses overspending.`;
}

export function leftToPlanText(left: Money, type: string, name: string): string {
  return `${formatMoneyText(left)} left to plan in ${levelWord(type)} '${name}', which refuses overspending.`;
}

export function lastChangedText(name: string, date: string): string {
  return `Last changed by ${name} on ${date}.`;
}
