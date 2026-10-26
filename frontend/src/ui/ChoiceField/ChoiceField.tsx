import type { ReactNode } from 'react';
import { formatWhole } from '@/format/decimal';
import { plural } from '@/format/text';
import { cx } from '../cx';
import { Combobox } from '../Combobox/Combobox';
import { FieldShell } from '../FieldShell/FieldShell';
import type { FieldFrameProps } from '../FieldShell/FieldShell';
import inputs from '../FieldShell/inputs.module.css';
import styles from './ChoiceField.module.css';

/** One choice. An inactive one is offered only while it is already chosen. */
export interface ChoiceOption {
  id: string;
  label: string;
  active: boolean;
}

interface ChoiceFieldBase extends FieldFrameProps {
  /** The field's name and the control's id, so FormErrorSummary can link to it. */
  name: string;
  /** Every option in the order to show them, inactive ones included. */
  options: ChoiceOption[];
  /** Locks the field and says why under it ("Choose a cluster first."). */
  disabledReason?: string;
  /** A neutral note under the field. */
  note?: ReactNode;
}

export interface SingleChoiceFieldProps extends ChoiceFieldBase {
  multiple?: false;
  /** The chosen option's id; "" for none. */
  value: string;
  onChange: (value: string) => void;
}

export interface MultipleChoiceFieldProps extends ChoiceFieldBase {
  multiple: true;
  value: string[];
  onChange: (value: string[]) => void;
}

export type ChoiceFieldProps = SingleChoiceFieldProps | MultipleChoiceFieldProps;

/** Up to this many options are a select or check boxes; more are searchable (design system §5). */
const SEARCHABLE_ABOVE = 10;
/** The searchable list shows at most this many matches (SC-007: 5,000 options stay fast). */
const LIST_LIMIT = 50;

/**
 * A single or multiple choice (feature 002, FR-024, SC-007). Up to 10 options: a select whose
 * first entry is "Choose…", or a group of check boxes. More: a searchable list (Combobox) that
 * finds any part of a label, shows at most 50 matches and the chosen values as chips. Inactive
 * options are offered only while chosen, and read "<label> (inactive)".
 */
export function ChoiceField(props: ChoiceFieldProps) {
  const { name, options, disabledReason, note, error, ...frameAndValue } = props;
  const chosen = props.multiple ? props.value : props.value ? [props.value] : [];
  const offered = options
    .filter((option) => option.active || chosen.includes(option.id))
    .map((option) => ({
      id: option.id,
      label: option.active ? option.label : `${option.label} (inactive)`,
    }));
  const frame = withoutValue(frameAndValue);

  if (offered.length > SEARCHABLE_ABOVE) {
    return (
      <Combobox
        {...frame}
        name={name}
        error={error}
        options={offered}
        value={chosen.filter((id) => offered.some((option) => option.id === id))}
        onChange={(ids) =>
          props.multiple ? props.onChange(ids) : props.onChange(ids[ids.length - 1] ?? '')
        }
        multiple={props.multiple}
        placeholder="Type to search"
        countText={(count) => plural(count, 'option')}
        noMatchText={(query) => `No option matches '${query}'.`}
        limit={LIST_LIMIT}
        limitText={(shown, total) =>
          `Showing ${formatWhole(shown)} of ${formatWhole(total)} — keep typing to narrow down.`
        }
        disabledReason={disabledReason}
        note={note}
      />
    );
  }

  if (props.multiple) {
    const { value, onChange } = props;
    return (
      <FieldShell {...frame} id={name} as="fieldset" error={error} note={disabledReason ?? note}>
        {() => (
          <div className={styles.boxes}>
            {offered.map((option, index) => {
              const id = `${name}-${index}`;
              return (
                <div key={option.id} className={styles.box}>
                  <input
                    id={id}
                    type="checkbox"
                    checked={value.includes(option.id)}
                    disabled={Boolean(disabledReason)}
                    onChange={(event) =>
                      onChange(
                        event.target.checked
                          ? [...value, option.id]
                          : value.filter((chosenId) => chosenId !== option.id),
                      )
                    }
                  />
                  <label htmlFor={id}>{option.label}</label>
                </div>
              );
            })}
          </div>
        )}
      </FieldShell>
    );
  }

  const { value, onChange } = props;
  return (
    <FieldShell {...frame} id={name} error={error} note={disabledReason ?? note}>
      {(describedBy) => (
        <select
          id={name}
          name={name}
          value={offered.some((option) => option.id === value) ? value : ''}
          disabled={Boolean(disabledReason)}
          onChange={(event) => onChange(event.target.value)}
          aria-invalid={error ? true : undefined}
          aria-describedby={describedBy}
          className={cx(inputs.input, inputs.select, error && inputs.invalid)}
        >
          <option value="">Choose…</option>
          {offered.map((option) => (
            <option key={option.id} value={option.id}>
              {option.label}
            </option>
          ))}
        </select>
      )}
    </FieldShell>
  );
}

/** The frame's props, without the value props of either variant. */
function withoutValue(
  props: Omit<ChoiceFieldProps, 'name' | 'options' | 'disabledReason' | 'note' | 'error'>,
): FieldFrameProps {
  const { label, marking, hint, tip, labelHidden } = props;
  return { label, marking, hint, tip, labelHidden };
}
