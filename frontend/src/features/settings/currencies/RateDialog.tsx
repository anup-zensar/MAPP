import { CircleAlert } from 'lucide-react';
import { useState } from 'react';
import { ApiError, conflictMessage } from '@/api/errors';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { DecimalField } from '@/ui/DecimalField/DecimalField';
import { FormDialog } from '@/ui/FormDialog/FormDialog';
import { problemOf } from '@/ui/FormPanel/problemOf';
import { LockedNote } from '@/ui/LockedNote/LockedNote';
import { useToast } from '@/ui/Toast/useToast';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import { inverseRate, parseDecimalInput } from '@/format/decimal';
import { localDate } from '@/format/intl';
import { useFormatters } from '@/format/useFormatters';
import { CONFLICT_NEXT_STEP } from '../save-failure';
import settingsStyles from '@/ui/settings-table.module.css';
import { inverseLine, rateLine } from './currencyActions';
import {
  checkAddRateImpact,
  checkUpdateRateImpact,
  fetchRate,
  useAddRate,
  useUpdateRate,
} from './ratesApi';
import type { ChangeImpact, ExchangeRateRecord } from './ratesApi';
import { futureRateNote } from './rateWords';
import styles from './rates.module.css';

/** Why a rate keeps its date (data-model §2.1). */
const DATE_FIXED = "A rate's date can't change: add a rate on the new date and delete this one.";

export interface RateDialogProps {
  /** The currency the rate is of, such as "EUR". */
  code: string;
  /** The rate to change; without one, the dialog adds a rate. */
  rate?: ExchangeRateRecord;
  onClose: () => void;
}

/** A save that waits for "Save rate" in the impact question. */
interface PendingSave {
  warnings: ChangeImpact['warnings'];
}

/**
 * Add rate and Edit rate (016 FR-003 – FR-005): the date (today unless changed; fixed once saved)
 * and the rate per 1 USD with the live line "1 USD = 0.95 EUR · 1 EUR = 1.052632 USD". A rate that
 * changes the current rate first says what it recalculates — line items, and plans planned in the
 * currency, before → after — with **Save rate** or **Keep editing**. A future rate says from when it
 * counts.
 */
export function RateDialog({ code, rate, onClose }: RateDialogProps) {
  const { show } = useToast();
  const { date: formatDate, dateFormat } = useFormatters();
  const addRate = useAddRate();
  const updateRate = useUpdateRate();
  const today = localDate(new Date());
  const [day, setDay] = useState(rate?.date ?? today);
  const [typed, setTyped] = useState(rate?.rate ?? '');
  const [version, setVersion] = useState(rate?.version ?? 0);
  const [saving, setSaving] = useState(false);
  const [problems, setProblems] = useState<FieldProblem[]>([]);
  const [failure, setFailure] = useState<{ message: string; nextStep?: string } | null>(null);
  const [pending, setPending] = useState<PendingSave | null>(null);
  const editing = rate !== undefined;
  const rateDay = rate?.date ?? day;

  async function submit() {
    setSaving(true);
    setFailure(null);
    try {
      const value = rateText(typed);
      const impact = editing
        ? await checkUpdateRateImpact(rate.id, { rate: value, version })
        : await checkAddRateImpact(code, { date: day, rate: value });
      setProblems([]);
      if (impact.warnings.length > 0) {
        setPending({ warnings: impact.warnings });
        return;
      }
      await save();
    } catch (error) {
      await showFailure(error);
    } finally {
      setSaving(false);
    }
  }

  async function save() {
    const value = rateText(typed);
    const saved = editing
      ? await updateRate.mutateAsync({ rateId: rate.id, body: { rate: value, version } })
      : await addRate.mutateAsync({ code, body: { date: day, rate: value } });
    show({ message: `Rate of ${code} on ${formatDate(saved.date)} saved` });
    onClose();
  }

  async function confirm() {
    setSaving(true);
    try {
      await save();
    } catch (error) {
      await showFailure(error);
    } finally {
      setSaving(false);
      setPending(null);
    }
  }

  async function showFailure(error: unknown) {
    const apiError = error instanceof ApiError ? error : new ApiError('other');
    if (apiError.kind === 'signInRequired' || apiError.kind === 'sessionExpired') {
      return;
    }
    if (apiError.kind === 'validation') {
      setProblems(apiError.fields);
      // A problem with no field of this dialog is shown above the fields.
      const other = apiError.fields.filter(
        (field) => field.field !== 'date' && field.field !== 'rate',
      );
      setFailure(other.length > 0 ? { message: other[0].message } : null);
      return;
    }
    setProblems([]);
    if (apiError.kind === 'conflict' && rate) {
      setFailure({
        message: conflictMessage(apiError, 'rate', dateFormat),
        nextStep: CONFLICT_NEXT_STEP,
      });
      // The typed rate stays; the next Save replaces the other person's change on purpose.
      const latest = await fetchRate(code, rate.id).catch(() => undefined);
      if (latest) {
        setVersion(latest.version);
      }
      return;
    }
    setFailure({ message: apiError.message });
  }

  const dateError = problemOf(problems, 'date');
  const future = rateDay > today;

  return (
    <>
      <FormDialog
        open
        onOpenChange={(open) => !open && onClose()}
        title={editing ? `Edit rate of ${code}` : `Add rate of ${code}`}
        submitLabel="Save rate"
        busy={saving}
        onSubmit={() => void submit()}
      >
        {failure && (
          <div role="alert" className={styles.failure}>
            <p>{failure.message}</p>
            {failure.nextStep && <p>{failure.nextStep}</p>}
          </div>
        )}
        {editing ? (
          <dl className={styles.fixedValue}>
            <dt>Date</dt>
            <dd>{formatDate(rate.date)}</dd>
            <dd>
              <LockedNote>{DATE_FIXED}</LockedNote>
            </dd>
          </dl>
        ) : (
          <div className={styles.field}>
            <label htmlFor="rate-date" className={styles.label}>
              Date<span className={styles.marking}>Required</span>
            </label>
            <input
              id="rate-date"
              name="date"
              type="date"
              value={day}
              onChange={(event) => setDay(event.target.value)}
              aria-invalid={dateError ? true : undefined}
              aria-describedby={
                [future && 'rate-date-note', dateError && 'rate-date-error']
                  .filter(Boolean)
                  .join(' ') || undefined
              }
              className={`${styles.input} ${dateError ? styles.invalid : ''}`}
            />
            {future && (
              <p id="rate-date-note" className={styles.note}>
                {futureRateNote(day, formatDate)}
              </p>
            )}
            {dateError && (
              <p id="rate-date-error" className={styles.error}>
                <CircleAlert aria-hidden="true" />
                {dateError}
              </p>
            )}
          </div>
        )}
        <DecimalField
          label="Rate (units per 1 USD)"
          name="rate"
          marking="Required"
          decimals={6}
          suffix="per USD"
          value={typed}
          onChange={setTyped}
          note={liveRate(code, typed)}
          error={problemOf(problems, 'rate')}
        />
        {editing && future && (
          <p className={settingsStyles.meta}>{futureRateNote(rate.date, formatDate)}</p>
        )}
      </FormDialog>
      {pending && (
        <ConfirmDialog
          open
          onOpenChange={(open) => !open && setPending(null)}
          title={`Save the rate of ${code} on ${formatDate(rateDay)}?`}
          confirmLabel="Save rate"
          cancelLabel="Keep editing"
          busy={saving}
          onConfirm={() => void confirm()}
        >
          {pending.warnings.length === 1 ? (
            <p>{pending.warnings[0].message}</p>
          ) : (
            <ul className={settingsStyles.warnings}>
              {pending.warnings.map((warning) => (
                <li key={warning.message}>{warning.message}</li>
              ))}
            </ul>
          )}
        </ConfirmDialog>
      )}
    </>
  );
}

/** "1 USD = 0.95 EUR · 1 EUR = 1.052632 USD" for a typed rate above 0; nothing otherwise. */
function liveRate(code: string, typed: string): string | undefined {
  const rate = parseDecimalInput(typed, 6);
  const inverse = rate === null ? null : inverseRate(rate);
  if (rate === null || inverse === null) {
    return undefined;
  }
  return `${rateLine(code, rate)} · ${inverseLine(code, inverse)}`;
}

/** A typed rate as the API takes it: grouping removed; anything else as typed, for the API to judge. */
function rateText(text: string): string {
  return parseDecimalInput(text, 6) ?? text.trim();
}
