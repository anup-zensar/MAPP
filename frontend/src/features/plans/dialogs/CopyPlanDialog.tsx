import { CircleAlert } from 'lucide-react';
import { useState } from 'react';
import { ApiError } from '@/api/errors';
import {
  CheckboxField,
  ErrorState,
  FailureMessage,
  FormDialog,
  LockedNote,
  Skeleton,
  TextField,
  problemOf,
} from '@/components';
import type { FieldProblem } from '@/components';
import { useCopyOptions } from '../plansApi';
import type { PlanCopyOptions, PlanCopyRequest } from '../plansApi';
import { contentsText } from '../planText';
import styles from './dialogs.module.css';

/** Why the API did not copy: problems for the fields, or a message for the whole form. */
export interface CopyRefusal {
  problems: FieldProblem[];
  message?: string;
}

export interface CopyPlanDialogProps {
  open: boolean;
  planId: string;
  planName: string;
  /**
   * Copies with the chosen name, year and contents. Answers why it was refused, or null when the
   * plan was copied (the caller then closes the dialog and opens the copy).
   */
  onCopy: (request: PlanCopyRequest) => Promise<CopyRefusal | null>;
  onClose: () => void;
}

/** Fields the dialog shows a problem under; any other problem is shown above the form. */
const DIALOG_FIELDS = ['name', 'year', 'includeContents'];

/**
 * Copy a plan (FR-026 – FR-028): the copy's name (suggested "<name> (copy)" or the next free
 * "(copy n)", editable), the same or the next year (unavailable with the reason when the next year
 * is not a permitted plan year), and whether everything inside comes along, with the counts. Give
 * it a new `key` each time it opens, so it starts from the suggestion.
 */
export function CopyPlanDialog({ open, planId, planName, onCopy, onClose }: CopyPlanDialogProps) {
  const options = useCopyOptions(planId, open);
  // What the person changed; until then the dialog shows the API's suggestion. The suggestion is
  // asked again on every opening, so it may arrive after the cached one and replace it.
  const [name, setName] = useState<string | null>(null);
  const [year, setYear] = useState<number | null>(null);
  const [includeContents, setIncludeContents] = useState(true);
  const [refusal, setRefusal] = useState<CopyRefusal | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit(loaded: PlanCopyOptions) {
    setBusy(true);
    try {
      setRefusal(
        await onCopy({
          name: name ?? loaded.name,
          year: year ?? loaded.sameYear,
          includeContents,
        }),
      );
    } finally {
      setBusy(false);
    }
  }

  function content() {
    if (options.data) {
      return (
        <CopyFields
          options={options.data}
          name={name ?? options.data.name}
          year={year ?? options.data.sameYear}
          includeContents={includeContents}
          refusal={refusal}
          onNameChange={setName}
          onYearChange={setYear}
          onIncludeContentsChange={setIncludeContents}
        />
      );
    }
    if (options.isError) {
      // A plan that is gone stays gone: its message alone, and Cancel closes the dialog.
      return options.error instanceof ApiError && options.error.kind === 'notFound' ? (
        <p role="alert" className={styles.problem}>
          <CircleAlert aria-hidden="true" />
          {options.error.message}
        </p>
      ) : (
        <ErrorState error={options.error} onRetry={() => void options.refetch()} />
      );
    }
    return <Skeleton lines={4} />;
  }

  return (
    <FormDialog
      open={open}
      onOpenChange={(next) => !next && onClose()}
      title={`Copy plan '${planName}'`}
      submitLabel="Copy plan"
      // Nothing to copy with until the options are there; the dialog says why meanwhile.
      onSubmit={() => options.data && void submit(options.data)}
      busy={busy}
    >
      {content()}
    </FormDialog>
  );
}

interface CopyFieldsProps {
  options: PlanCopyOptions;
  name: string;
  year: number;
  includeContents: boolean;
  refusal: CopyRefusal | null;
  onNameChange: (name: string) => void;
  onYearChange: (year: number) => void;
  onIncludeContentsChange: (include: boolean) => void;
}

/** The dialog's fields, once the options are loaded. */
function CopyFields({
  options,
  name,
  year,
  includeContents,
  refusal,
  onNameChange,
  onYearChange,
  onIncludeContentsChange,
}: CopyFieldsProps) {
  const problems = refusal?.problems ?? [];
  const otherProblems = problems.filter((problem) => !DIALOG_FIELDS.includes(problem.field));
  const failure = refusal?.message ?? otherProblems.map((problem) => problem.message).join(' ');
  const counts = contentsText(options.contents);
  const nextYear = options.nextYear;
  const yearError = problemOf(problems, 'year');

  return (
    // The fields arrive after the dialog has opened, so the name takes the focus when it appears
    // (FormDialog had put it on Copy plan while the options loaded).
    <div className={styles.stack} ref={focusNameOnArrival}>
      {failure && <FailureMessage failure={{ message: failure }} />}
      <TextField
        label="Name of the copy"
        name="name"
        marking="Required"
        maxLength={120}
        value={name}
        onChange={onNameChange}
        error={problemOf(problems, 'name')}
      />
      <fieldset
        className={styles.radioGroup}
        aria-describedby={yearError ? 'year-error' : undefined}
      >
        <legend className={styles.legend}>Year</legend>
        <label className={styles.radio}>
          <input
            type="radio"
            name="year"
            value={options.sameYear}
            checked={year === options.sameYear}
            onChange={() => onYearChange(options.sameYear)}
          />
          Same year ({options.sameYear})
        </label>
        <div>
          <label className={styles.radio}>
            <input
              type="radio"
              name="year"
              value={nextYear.year}
              checked={year === nextYear.year}
              disabled={!nextYear.available}
              aria-describedby={nextYear.available ? undefined : 'next-year-reason'}
              onChange={() => onYearChange(nextYear.year)}
            />
            Next year ({nextYear.year})
          </label>
          {!nextYear.available && (
            <div className={styles.radioNote}>
              <LockedNote id="next-year-reason">{nextYear.reason}</LockedNote>
            </div>
          )}
        </div>
        {year === nextYear.year && <p className={styles.note}>Every date moves one year later.</p>}
        {yearError && (
          <p id="year-error" className={styles.problem}>
            <CircleAlert aria-hidden="true" />
            {yearError}
          </p>
        )}
      </fieldset>
      <CheckboxField
        label="Include campaigns, initiatives, activities and line items"
        name="includeContents"
        checked={includeContents}
        onChange={onIncludeContentsChange}
        hint={counts || 'This plan has nothing inside yet.'}
        error={problemOf(problems, 'includeContents')}
      />
    </div>
  );
}

/** Moves the focus to the name field, unless the person already moved it into the form. */
function focusNameOnArrival(node: HTMLDivElement | null) {
  const active = document.activeElement;
  if (node && !(active instanceof HTMLInputElement)) {
    node.querySelector<HTMLInputElement>('input[name="name"]')?.focus();
  }
}
