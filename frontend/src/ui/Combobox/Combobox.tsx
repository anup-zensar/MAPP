import { Check, X } from 'lucide-react';
import { Popover } from 'radix-ui';
import { useEffect, useId, useRef, useState } from 'react';
import type { KeyboardEvent, ReactNode } from 'react';
import { cx } from '../cx';
import { FieldShell } from '../FieldShell/FieldShell';
import type { FieldFrameProps } from '../FieldShell/FieldShell';
import inputs from '../FieldShell/inputs.module.css';
import { InfoTip } from '../InfoTip/InfoTip';
import styles from './Combobox.module.css';

/** One choice of the list. */
export interface ComboboxOption {
  id: string;
  /** What it is called: in chips, in "Remove <label>" and, without `content`, in the list. */
  label: string;
  /** What a search looks in, ignoring case ("name e-mail job title"); the label when left out. */
  searchText?: string;
  /** What the list shows for it when that is more than the label (an avatar, a second line). */
  content?: ReactNode;
}

/** How a chosen value looks as a chip. */
export interface ComboboxChip {
  /** Names the chip's Remove button: "Remove <label>". */
  label: string;
  /** What the chip shows; the label when left out. */
  content?: ReactNode;
  /** A short explanation in an ⓘ on the chip, for example why an inactive value is kept. */
  tip?: string;
}

export interface ComboboxProps extends FieldFrameProps {
  /** The field's name and the input's id, so FormErrorSummary can link to it. */
  name: string;
  /** What can be chosen, in the order to list them. Chosen values need not be among them. */
  options: ComboboxOption[];
  /** The chosen ids, in the order chosen. */
  value: string[];
  onChange: (ids: string[]) => void;
  /** Several values (chosen ones stay listed, ticked); otherwise one, which a new choice replaces. */
  multiple?: boolean;
  /** How a chosen id looks as a chip; by default the option's label (or the id). */
  renderChip?: (id: string) => ComboboxChip;
  placeholder?: string;
  /** The count of matching options, shown above the list and told to screen readers: "300 people". */
  countText: (count: number) => string;
  /** Said when nothing matches: "No one matches 'xyz'." */
  noMatchText: (query: string) => string;
  /** Lists at most this many matches (for very long lists, such as 5,000 options). */
  limit?: number;
  /** Said under a list cut at `limit`: "Showing 50 of 1,217 — keep typing to narrow down." */
  limitText?: (shown: number, total: number) => string;
  /** Locks the field and says why under it; chosen values stay visible. */
  disabledReason?: string;
  /** A neutral note under the field. */
  note?: ReactNode;
}

/**
 * The one searchable choice of MAPP (feature 003 plan, ARIA 1.2 combobox): an input that
 * filters the options by any part of their search text, a list below it, and the chosen values as
 * chips with a Remove button. ↓ ↑ move through the list, Enter chooses, Escape closes it and
 * Backspace in an empty input removes the last chip. The focus always stays in the input.
 */
export function Combobox({
  name,
  options,
  value,
  onChange,
  multiple = false,
  renderChip,
  placeholder,
  countText,
  noMatchText,
  limit,
  limitText,
  disabledReason,
  note,
  error,
  ...frame
}: ComboboxProps) {
  const [query, setQuery] = useState('');
  const [open, setOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState(0);
  const anchor = useRef<HTMLDivElement>(null);
  const input = useRef<HTMLInputElement>(null);
  const listId = useId();
  const disabled = Boolean(disabledReason);

  const search = query.trim().toLowerCase();
  const matches = search
    ? options.filter((option) => (option.searchText ?? option.label).toLowerCase().includes(search))
    : options;
  const shown = limit === undefined ? matches : matches.slice(0, limit);
  const active = shown.length === 0 ? -1 : Math.min(activeIndex, shown.length - 1);
  const optionId = (index: number) => `${listId}-option-${index}`;
  const isOpen = open && !disabled;

  useEffect(() => {
    // Keeps the option chosen with the arrow keys in view inside the scrolling list.
    if (isOpen && active >= 0) {
      document.getElementById(`${listId}-option-${active}`)?.scrollIntoView({ block: 'nearest' });
    }
  }, [isOpen, active, listId]);

  function chipOf(id: string): ComboboxChip {
    if (renderChip) {
      return renderChip(id);
    }
    return { label: options.find((option) => option.id === id)?.label ?? id };
  }

  function choose(id: string) {
    if (multiple) {
      onChange(value.includes(id) ? value.filter((chosen) => chosen !== id) : [...value, id]);
    } else {
      onChange([id]);
      setOpen(false);
    }
    setQuery('');
    setActiveIndex(0);
  }

  function remove(id: string) {
    onChange(value.filter((chosen) => chosen !== id));
    input.current?.focus();
  }

  function onKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (disabled) {
      return;
    }
    switch (event.key) {
      case 'ArrowDown':
        event.preventDefault();
        if (!isOpen) {
          setOpen(true);
          setActiveIndex(0);
        } else {
          setActiveIndex(Math.min(active + 1, shown.length - 1));
        }
        break;
      case 'ArrowUp':
        event.preventDefault();
        setOpen(true);
        setActiveIndex(Math.max(active - 1, 0));
        break;
      case 'Enter':
        // Closed, Enter submits the form as in any field; open, it chooses.
        if (isOpen && active >= 0) {
          event.preventDefault();
          choose(shown[active].id);
        }
        break;
      case 'Backspace':
        if (multiple && query === '' && value.length > 0) {
          onChange(value.slice(0, -1));
        }
        break;
    }
  }

  const status = isOpen ? countText(matches.length) : '';

  return (
    <FieldShell id={name} error={error} note={disabledReason ?? note} {...frame}>
      {(describedBy) => (
        <Popover.Root open={isOpen} onOpenChange={setOpen}>
          <Popover.Anchor asChild>
            <div
              ref={anchor}
              className={cx(styles.control, error && inputs.invalid, disabled && styles.disabled)}
              onClick={() => input.current?.focus()}
            >
              {value.map((id) => {
                const chip = chipOf(id);
                return (
                  <span key={id} className={styles.chip}>
                    <span className={styles.chipContent}>{chip.content ?? chip.label}</span>
                    {chip.tip && <InfoTip label={chip.label} text={chip.tip} />}
                    {!disabled && (
                      <button
                        type="button"
                        className={styles.remove}
                        aria-label={`Remove ${chip.label}`}
                        onClick={(event) => {
                          event.stopPropagation();
                          remove(id);
                        }}
                      >
                        <X />
                      </button>
                    )}
                  </span>
                );
              })}
              <input
                ref={input}
                id={name}
                name={name}
                type="text"
                role="combobox"
                autoComplete="off"
                aria-autocomplete="list"
                aria-expanded={isOpen}
                aria-controls={isOpen && shown.length > 0 ? listId : undefined}
                aria-activedescendant={isOpen && active >= 0 ? optionId(active) : undefined}
                aria-disabled={disabled || undefined}
                aria-invalid={error ? true : undefined}
                aria-describedby={describedBy}
                readOnly={disabled}
                placeholder={value.length === 0 ? placeholder : undefined}
                value={query}
                onChange={(event) => {
                  setQuery(event.target.value);
                  setActiveIndex(0);
                  setOpen(true);
                }}
                onClick={() => setOpen(true)}
                onKeyDown={onKeyDown}
                onBlur={() => setOpen(false)}
                className={styles.input}
              />
            </div>
          </Popover.Anchor>
          <span role="status" className="visually-hidden">
            {status}
          </span>
          <Popover.Portal>
            <Popover.Content
              role="presentation"
              className={styles.popover}
              align="start"
              sideOffset={4}
              style={{ width: 'var(--radix-popover-trigger-width)' }}
              // The focus stays in the input: it never moves into the list.
              onOpenAutoFocus={(event) => event.preventDefault()}
              onCloseAutoFocus={(event) => event.preventDefault()}
              onInteractOutside={(event) => {
                if (anchor.current?.contains(event.target as Node)) {
                  event.preventDefault();
                }
              }}
              // Keeps the focus in the input when an option or the list's scroll bar is clicked.
              onMouseDown={(event) => event.preventDefault()}
            >
              {shown.length === 0 ? (
                <p className={styles.message}>{noMatchText(query.trim())}</p>
              ) : (
                <>
                  <p className={styles.count} aria-hidden="true">
                    {countText(matches.length)}
                  </p>
                  <ul
                    id={listId}
                    role="listbox"
                    aria-label={frame.label}
                    aria-multiselectable={multiple || undefined}
                    className={styles.list}
                  >
                    {shown.map((option, index) => {
                      const chosen = value.includes(option.id);
                      return (
                        <li
                          key={option.id}
                          id={optionId(index)}
                          role="option"
                          aria-selected={chosen}
                          className={cx(styles.option, index === active && styles.active)}
                          onClick={() => choose(option.id)}
                        >
                          <span className={styles.optionContent}>
                            {option.content ?? option.label}
                          </span>
                          {chosen && <Check className={styles.check} aria-hidden="true" />}
                        </li>
                      );
                    })}
                  </ul>
                  {limitText && shown.length < matches.length && (
                    <p className={styles.message}>{limitText(shown.length, matches.length)}</p>
                  )}
                </>
              )}
            </Popover.Content>
          </Popover.Portal>
        </Popover.Root>
      )}
    </FieldShell>
  );
}
