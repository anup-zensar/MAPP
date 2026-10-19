import { ArrowDown, ArrowUp, Plus, X } from 'lucide-react';
import { useEffect, useRef } from 'react';
import type { ReactNode } from 'react';
import type { components } from '@/api/schema';
import { Button, problemOf, TextField, Tooltip } from '@/components';
import type { FieldProblem } from '@/components';
import { LinkedToField } from './LinkedToField';
import styles from './ShellAttributes.module.css';

type FormAttribute = components['schemas']['FormAttribute'];
type FormOption = components['schemas']['FormOption'];

/** One option being typed; `key` keeps the row's field while rows move. */
export interface OptionRow {
  key: number;
  label: string;
  /** The parent options it is linked to (dependent attributes only). */
  parentOptionIds: string[];
}

export interface OptionsEditorProps {
  rows: OptionRow[];
  onChange: (rows: OptionRow[]) => void;
  /** The problems of the last save: "options", "options[2].label", "options[2].parentOptionIds". */
  problems: FieldProblem[];
  /** The attribute the new attribute depends on: each option then gets its Linked to field. */
  parent?: FormAttribute;
}

/** The id of a row's field, which is also the API's name for it, so the summary links to it. */
function fieldName(index: number): string {
  return `options[${index}].label`;
}

/**
 * The first options of a new choice attribute (feature 002, FR-018, FR-022): one field per option
 * in the order of the form, with Move up, Move down and Remove, and **Add option** at the end. For
 * a dependent attribute each option also says which parent options it belongs to (Linked to).
 */
export function OptionsEditor({ rows, onChange, problems, parent }: OptionsEditorProps) {
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
    onChange([...rows, { key, label: '', parentOptionIds: [] }]);
  }

  function move(index: number, by: -1 | 1) {
    const next = [...rows];
    [next[index], next[index + by]] = [next[index + by], next[index]];
    onChange(next);
  }

  function update(key: number, change: Partial<OptionRow>) {
    onChange(rows.map((each) => (each.key === key ? { ...each, ...change } : each)));
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
                <div
                  role="group"
                  aria-label={`Option ${index + 1}`}
                  className={styles.optionFields}
                >
                  <TextField
                    label={`Option ${index + 1}`}
                    name={fieldName(index)}
                    maxLength={120}
                    value={row.label}
                    onChange={(label) => update(row.key, { label })}
                    error={problemOf(problems, fieldName(index))}
                  />
                  {parent && (
                    <LinkedToField
                      name={`options[${index}].parentOptionIds`}
                      parent={parent}
                      value={row.parentOptionIds}
                      onChange={(parentOptionIds) => update(row.key, { parentOptionIds })}
                      error={problemOf(problems, `options[${index}].parentOptionIds`)}
                    />
                  )}
                </div>
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

export interface OptionLinksEditorProps {
  /** The attribute's active options, in display order. */
  options: FormOption[];
  /** The attribute it will depend on. */
  parent: FormAttribute;
  /** The parent options chosen for each option, by option id. */
  links: Record<string, string[]>;
  onChange: (links: Record<string, string[]>) => void;
  /** The problems of the last save: "optionLinks.SO-1012". */
  problems: FieldProblem[];
}

/**
 * When an attribute starts depending on another one (edit form, FR-022): every active option with
 * its Linked to field, because each one needs at least one option of the new parent before saving.
 */
export function OptionLinksEditor({
  options,
  parent,
  links,
  onChange,
  problems,
}: OptionLinksEditorProps) {
  return (
    <ol role="list" className={styles.optionRows}>
      {options.map((option) => {
        const name = `optionLinks.${option.id}`;
        return (
          <li key={option.id} role="group" aria-label={option.label} className={styles.linkRow}>
            <p className={styles.linkLabel}>{option.label}</p>
            <LinkedToField
              name={name}
              parent={parent}
              value={links[option.id] ?? []}
              onChange={(ids) => onChange({ ...links, [option.id]: ids })}
              error={problemOf(problems, name)}
            />
          </li>
        );
      })}
    </ol>
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
