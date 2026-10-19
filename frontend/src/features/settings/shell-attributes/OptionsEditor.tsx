import { ArrowDown, ArrowUp, Plus, X } from 'lucide-react';
import { useEffect, useRef } from 'react';
import type { ReactNode } from 'react';
import { Button, problemOf, TextField, Tooltip } from '@/components';
import type { FieldProblem } from '@/components';
import styles from './ShellAttributes.module.css';

/** One option being typed; `key` keeps the row's field while rows move. */
export interface OptionRow {
  key: number;
  label: string;
}

export interface OptionsEditorProps {
  rows: OptionRow[];
  onChange: (rows: OptionRow[]) => void;
  /** The problems of the last save: "options" and "options[2].label". */
  problems: FieldProblem[];
}

/** The id of a row's field, which is also the API's name for it, so the summary links to it. */
function fieldName(index: number): string {
  return `options[${index}].label`;
}

/**
 * The first options of a new choice attribute (feature 002, FR-018): one field per option in the
 * order of the form, with Move up, Move down and Remove, and **Add option** at the end.
 */
export function OptionsEditor({ rows, onChange, problems }: OptionsEditorProps) {
  // The row to put the focus in once it is on screen (after Add option).
  const focusRow = useRef<number | null>(null);

  useEffect(() => {
    if (focusRow.current !== null) {
      document.getElementById(fieldName(focusRow.current))?.focus();
      focusRow.current = null;
    }
  }, [rows.length]);

  function add() {
    const key = rows.reduce((highest, row) => Math.max(highest, row.key), 0) + 1;
    focusRow.current = rows.length;
    onChange([...rows, { key, label: '' }]);
  }

  function move(index: number, by: -1 | 1) {
    const next = [...rows];
    [next[index], next[index + by]] = [next[index + by], next[index]];
    onChange(next);
  }

  const optionsProblem = problemOf(problems, 'options');

  return (
    <>
      {rows.length > 0 && (
        <ol role="list" className={styles.optionRows}>
          {rows.map((row, index) => {
            const name = row.label.trim() || `option ${index + 1}`;
            return (
              <li key={row.key} className={styles.optionRow}>
                <TextField
                  label={`Option ${index + 1}`}
                  name={fieldName(index)}
                  maxLength={120}
                  value={row.label}
                  onChange={(label) =>
                    onChange(rows.map((each) => (each.key === row.key ? { ...each, label } : each)))
                  }
                  error={problemOf(problems, fieldName(index))}
                />
                <div className={styles.optionButtons}>
                  <RowButton
                    label={`Move '${name}' up`}
                    icon={<ArrowUp />}
                    disabledReason={index === 0 ? `'${name}' is already first.` : undefined}
                    onClick={() => move(index, -1)}
                  />
                  <RowButton
                    label={`Move '${name}' down`}
                    icon={<ArrowDown />}
                    disabledReason={
                      index === rows.length - 1 ? `'${name}' is already last.` : undefined
                    }
                    onClick={() => move(index, 1)}
                  />
                  <RowButton
                    label={`Remove '${name}'`}
                    icon={<X />}
                    onClick={() => onChange(rows.filter((each) => each.key !== row.key))}
                  />
                </div>
              </li>
            );
          })}
        </ol>
      )}
      <div>
        <Button
          id="options"
          icon={<Plus />}
          onClick={add}
          aria-describedby={optionsProblem ? 'options-error' : undefined}
        >
          Add option
        </Button>
        {optionsProblem && (
          <p id="options-error" className={styles.problem}>
            {optionsProblem}
          </p>
        )}
      </div>
    </>
  );
}

interface RowButtonProps {
  label: string;
  icon: ReactNode;
  /** Why it can't be used now; it stays focusable and its tip says why. */
  disabledReason?: string;
  onClick: () => void;
}

/** An icon button of a row, named for the option it acts on. */
function RowButton({ label, icon, disabledReason, onClick }: RowButtonProps) {
  return (
    <Tooltip content={disabledReason ?? label}>
      <Button
        variant="ghost"
        size="sm"
        icon={icon}
        aria-label={label}
        aria-disabled={disabledReason ? true : undefined}
        className={disabledReason ? styles.unavailable : undefined}
        onClick={() => {
          if (!disabledReason) {
            onClick();
          }
        }}
      />
    </Tooltip>
  );
}
