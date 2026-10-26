import { plural } from '@/format/text';
import { activityMonths, isOutsideDates } from './line-item-math';
import type { ActivityDates, SpreadMethod } from './line-item-math';

export interface SpreadFacts {
  method: SpreadMethod;
  months: string[];
}

/** How many line items new activity dates re-spread, keep as typed or leave unspread (D-150). */
export interface SpreadEffect {
  respread: number;
  customKept: number;
  customOutside: number;
  notSpread: number;
}

export function spreadEffect(items: SpreadFacts[], dates: ActivityDates): SpreadEffect {
  const hasDates = activityMonths(dates).length > 0;
  const effect: SpreadEffect = { respread: 0, customKept: 0, customOutside: 0, notSpread: 0 };
  for (const item of items) {
    if (item.method === 'custom') {
      effect.customKept += 1;
      if (item.months.some((month) => isOutsideDates(month, dates))) {
        effect.customOutside += 1;
      }
    } else if (hasDates) {
      effect.respread += 1;
    } else {
      effect.notSpread += 1;
    }
  }
  return effect;
}

/** "When you save, 2 line items are re-spread over the new dates …" (FR-022); null for no change. */
export function spreadEffectText(effect: SpreadEffect): string | null {
  const sentences: string[] = [];
  const respread = effect.respread
    ? `${plural(effect.respread, 'line item')} ${isAre(effect.respread)} re-spread over the new dates`
    : null;
  const kept = effect.customKept
    ? `${plural(effect.customKept, 'custom spread')} ${isAre(effect.customKept)} kept as typed`
    : null;
  if (respread && kept) {
    sentences.push(`When you save, ${respread} and ${kept}.`);
  } else if (respread || kept) {
    sentences.push(`When you save, ${respread ?? kept}.`);
  }
  if (effect.customOutside) {
    sentences.push(
      `${plural(effect.customOutside, 'custom spread')} will have months outside the activity's dates.`,
    );
  }
  if (effect.notSpread) {
    sentences.push(
      `Without dates, ${plural(effect.notSpread, 'line item')} won't be spread until the activity has dates again.`,
    );
  }
  return sentences.length > 0 ? sentences.join(' ') : null;
}

function isAre(count: number): string {
  return count === 1 ? 'is' : 'are';
}
