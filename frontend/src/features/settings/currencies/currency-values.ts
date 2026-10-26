import { parseDecimalInput } from '@/format/decimal';
import type { Currency, CurrencyCreate, CurrencyUpdate } from './api';
import { lockOf } from './currency-actions';
import { rateText } from './rate-input';
import type { CurrencyValues } from './schemas';

export function startValues(currency: Currency | undefined): CurrencyValues {
  return {
    code: currency?.code ?? '',
    name: currency?.name ?? '',
    symbol: currency?.symbol ?? '',
    rate: currency?.rate ?? '',
  };
}

export function createBody(values: CurrencyValues): CurrencyCreate {
  return {
    code: values.code.trim(),
    name: values.name,
    symbol: values.symbol,
    rate: rateText(values.rate),
  };
}

/** The fields people changed, with the version (D-203); null when nothing changed. */
export function changedFields(
  currency: Currency,
  values: CurrencyValues,
  version: number,
): CurrencyUpdate | null {
  const body: CurrencyUpdate = { version };
  if (values.name !== currency.name) {
    body.name = values.name;
  }
  if (values.symbol !== currency.symbol) {
    body.symbol = values.symbol;
  }
  if (!lockOf(currency, 'rate') && !sameRate(values.rate, currency.rate)) {
    body.rate = rateText(values.rate);
  }
  return Object.keys(body).length > 1 ? body : null;
}

/** "0.920" and "0.92" are the same rate. */
function sameRate(typed: string, saved: string): boolean {
  const rate = parseDecimalInput(typed, 6);
  return rate !== null && trimZeros(rate) === trimZeros(saved);
}

function trimZeros(rate: string): string {
  return rate.includes('.') ? rate.replace(/\.?0+$/, '') : rate;
}
