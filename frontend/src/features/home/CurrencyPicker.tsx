import { useId } from 'react';
import { ApiError } from '@/api/errors';
import type { components } from '@/api/schema';
import { useCurrencies } from '@/api/settingsQueries';
import { useToast } from '@/components';
import { useFormatters } from '@/lib/useFormatters';
import { useSaveDisplayCurrency } from './homeApi';
import { CHOICE_NOT_SAVED } from './homeText';
import styles from './CurrencyPicker.module.css';

type Currency = components['schemas']['Currency'];

interface CurrencyOption {
  code: string;
  /** "EUR — Euro"; only the code while the list is loading or could not be loaded. */
  label: string;
}

/**
 * The active currencies (feature 002), plus the person's display currency when it has been set
 * inactive, because it stays in use (D-063, D-201).
 */
function currencyOptions(currencies: Currency[] | undefined, current: string): CurrencyOption[] {
  const options = (currencies ?? [])
    .filter((currency) => currency.active || currency.code === current)
    .map((currency) => ({ code: currency.code, label: `${currency.code} — ${currency.name}` }));
  return options.some((option) => option.code === current)
    ? options
    : [{ code: current, label: current }, ...options];
}

/**
 * Home's Currency picker (FR-002): starts on the person's display currency; a choice saves it as
 * their display currency, and every amount on Home is shown again in it. A failed save says so,
 * with Try again, and the picker goes back to the saved currency.
 */
export function CurrencyPicker() {
  const id = useId();
  const { displayCurrency } = useFormatters();
  const currencies = useCurrencies();
  const save = useSaveDisplayCurrency();
  const { show } = useToast();
  // The chosen currency shows at once; it goes back to the saved one when the save fails.
  const value = (save.isPending && save.variables) || displayCurrency;

  function choose(code: string) {
    save.mutate(code, {
      onError: (failure) => {
        // A session that has ended is handled by the session alone (001 plan "Session").
        if (
          failure instanceof ApiError &&
          (failure.kind === 'signInRequired' || failure.kind === 'sessionExpired')
        ) {
          return;
        }
        show({
          message: CHOICE_NOT_SAVED,
          action: { label: 'Try again', altText: 'Try again', onClick: () => choose(code) },
        });
      },
    });
  }

  return (
    <div className={styles.picker}>
      <label htmlFor={id} className={styles.label}>
        Currency
      </label>
      <select
        id={id}
        className={styles.select}
        value={value}
        onChange={(event) => choose(event.target.value)}
      >
        {currencyOptions(currencies.data?.items, value).map((option) => (
          <option key={option.code} value={option.code}>
            {option.label}
          </option>
        ))}
      </select>
    </div>
  );
}
