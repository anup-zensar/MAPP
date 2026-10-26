import { inverseRate, parseDecimalInput } from '@/format/decimal';
import { inverseLine, rateLine } from './currency-actions';

/** A typed rate as the API takes it: grouping removed; anything else as typed, for the API to judge. */
export function rateText(text: string): string {
  return parseDecimalInput(text, 6) ?? text.trim();
}

export function liveRate(code: string, typed: string): string | undefined {
  const rate = parseDecimalInput(typed, 6);
  const inverse = rate === null ? null : inverseRate(rate);
  if (rate === null || inverse === null || code === '') {
    return undefined;
  }
  return `${rateLine(code, rate)} · ${inverseLine(code, inverse)}`;
}
