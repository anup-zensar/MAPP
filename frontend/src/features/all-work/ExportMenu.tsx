import { Download, LoaderCircle } from 'lucide-react';
import { DropdownMenu } from 'radix-ui';
import { useId, useRef, useState } from 'react';
import { ApiError } from '@/api/errors';
import { Button } from '@/ui/Button/Button';
import { useToast } from '@/ui/Toast/useToast';
import type { AllWorkResult, ViewState } from './all-work-types';
import styles from './ExportMenu.module.css';
import { exportAllWork } from './api';

type Format = 'xlsx' | 'csv';

const FORMATS: { format: Format; label: string }[] = [
  { format: 'xlsx', label: 'Spreadsheet (.xlsx)' },
  { format: 'csv', label: 'CSV (.csv)' },
];

export interface ExportMenuProps {
  state: ViewState;
  result: AllWorkResult | undefined;
  /** Why the person may not export: both formats are shown disabled with it. */
  refusal?: string;
}

/**
 * The view on screen as a file the API writes with every row, the totals and a note of every
 * filter, saved under the name the API gives.
 */
export function ExportMenu({ state, result, refusal }: ExportMenuProps) {
  const { show } = useToast();
  const [preparing, setPreparing] = useState(false);
  const chosen = useRef<Format | null>(null);
  const hintId = useId();

  async function exportAs(format: Format) {
    if (result && result.rows.length === 0) {
      show({ message: 'Nothing to export — clear a filter first.' });
      return;
    }
    setPreparing(true);
    try {
      const fileName = await exportAllWork(state, format);
      show({ message: `Exported '${fileName}'` });
    } catch (error) {
      const failure = error instanceof ApiError ? error : new ApiError('other');
      if (failure.kind !== 'signInRequired' && failure.kind !== 'sessionExpired') {
        show({ message: failure.message });
      }
    } finally {
      setPreparing(false);
    }
  }

  return (
    <DropdownMenu.Root modal={false}>
      <DropdownMenu.Trigger asChild>
        <Button
          icon={preparing ? <LoaderCircle className={styles.spinner} /> : <Download />}
          aria-busy={preparing || undefined}
        >
          {preparing ? 'Preparing your file…' : 'Export'}
        </Button>
      </DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        <DropdownMenu.Content
          className={styles.menu}
          align="end"
          sideOffset={4}
          onCloseAutoFocus={() => {
            const format = chosen.current;
            chosen.current = null;
            if (format) {
              void exportAs(format);
            }
          }}
        >
          {FORMATS.map(({ format, label }) => (
            <DropdownMenu.Item
              key={format}
              className={styles.item}
              disabled={preparing || refusal !== undefined}
              aria-describedby={refusal ? hintId : undefined}
              onSelect={() => {
                chosen.current = format;
              }}
            >
              {label}
            </DropdownMenu.Item>
          ))}
          {refusal && (
            <p id={hintId} className={styles.hint}>
              {refusal}
            </p>
          )}
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  );
}
