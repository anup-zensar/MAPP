/**
 * The words of the exchange-rate screens (feature 016, US1; plan "User-visible messages"), in one
 * place so the details, the rate history, the dialog and the import say things the same way.
 */
import { formatWhole } from '@/format/decimal';
import { localDate } from '@/format/intl';
import type { Currency } from './api';
import type { RateImportCounts, RateImportOutcome, RateSource } from './rates-api';

/** "Entered by hand" or "GSAP import" (FR-001). */
export const SOURCE_WORDS: Record<RateSource, string> = {
  manual: 'Entered by hand',
  gsapImport: 'GSAP import',
};

/**
 * "Rate of 19 Oct 2026 · GSAP import" (US1-1): the date and the source of the rate conversions
 * use; just the date when the source is unknown.
 */
export function rateOfLine(
  rateDate: string,
  source: RateSource | null | undefined,
  date: (value: string) => string,
): string {
  const of = `Rate of ${date(rateDate)}`;
  return source ? `${of} · ${SOURCE_WORDS[source]}` : of;
}

/** "GSAP import · gsap_rates_2026-10-19.csv", or "Entered by hand" (FR-007). */
export function sourceOf(source: RateSource, importFileName: string | null): string {
  return importFileName ? `${SOURCE_WORDS[source]} · ${importFileName}` : SOURCE_WORDS[source];
}

/** The outcomes of a reviewed rate file, in the order the review lists them (FR-006). */
export const OUTCOMES: RateImportOutcome[] = ['problem', 'notUsed', 'replace', 'add', 'present'];

/** What each outcome is called in the review. */
export const OUTCOME_WORDS: Record<RateImportOutcome, string> = {
  add: 'To add',
  replace: 'Replaces a rate',
  present: 'Already present',
  notUsed: 'Not used',
  problem: 'Problems',
};

/** The rows an import saves: the rows to add and those that replace a rate. */
export function rowsToImport(counts: RateImportCounts): number {
  return counts.add + counts.replace;
}

/** "Import 89 rates", "Import 1 rate". */
export function importLabel(count: number): string {
  return `Import ${formatWhole(count)} ${count === 1 ? 'rate' : 'rates'}`;
}

/** "89 exchange rates imported from 'gsap_rates_2026-10-20.csv'." (plan, toast). */
export function importedMessage(count: number, fileName: string): string {
  const rates = count === 1 ? 'exchange rate' : 'exchange rates';
  return `${formatWhole(count)} ${rates} imported from '${fileName}'.`;
}

/** Nothing in the file would change a rate (plan "User-visible messages"). */
export const NOTHING_TO_IMPORT = 'Nothing to import: every rate is already present or not used.';

/** The largest file the import takes (FR-006). */
export const MAX_FILE_BYTES = 2 * 1024 * 1024;

export const FILE_TOO_LARGE = 'The file is larger than 2 MB. Split it and import each part.';

export const NOT_A_CSV = 'Choose a .csv file: the GSAP daily rate extract.';

/** "No conversion changes before 01 Jan 2027." for a rate dated after today (data-model §3.2). */
export function futureRateNote(rateDate: string, date: (value: string) => string): string {
  return `No conversion changes before ${date(rateDate)}.`;
}

/** "EUR needs a rate dated today or earlier." — the last such rate can't be deleted (FR-003). */
export function lastRateReason(code: string): string {
  return `${code} needs a rate dated today or earlier.`;
}

/**
 * "Rate of 19 Oct 2026 · GSAP import" (016 FR-003); an API without dated rates yet answers who
 * changed the rate last and when.
 */
export function rateSourceLine(currency: Currency, date: (value: string) => string): string {
  if (currency.rateDate) {
    return rateOfLine(currency.rateDate, currency.rateSource, date);
  }
  return `Rate last changed by ${currency.rateChangedBy.name} on ${date(localDate(currency.rateChangedAt))}`;
}

/** The date of a currency's current rate; an API without dated rates yet answers when it changed. */
export function rateDateOf(currency: Currency): string {
  return currency.rateDate ?? localDate(currency.rateChangedAt);
}
