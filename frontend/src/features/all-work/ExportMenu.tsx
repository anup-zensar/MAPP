import { Download, LoaderCircle } from 'lucide-react';
import { DropdownMenu } from 'radix-ui';
import { useRef, useState } from 'react';
import { api } from '@/api/client';
import { ApiError } from '@/api/errors';
import { Button, useToast } from '@/components';
import { downloadFromApi } from '@/lib/download';
import type { AllWorkResult, ViewState } from './allWorkTypes';
import styles from './ExportMenu.module.css';
import { toQueryParams } from './viewState';

type Format = 'xlsx' | 'csv';

export interface ExportMenuProps {
  /** The view on screen: the file holds its rows and columns as filtered (FR-032). */
  state: ViewState;
  /** The answer on screen; undefined while it loads. */
  result: AllWorkResult | undefined;
}

/**
 * **Export** (US7, FR-032 – FR-034): the current view as a spreadsheet or a CSV file, written by
 * the API with every row (collapsed branches too), the totals and a note of every filter, and
 * saved under the name the API gives ("MAPP All work – Grid – 2026 – 18 Oct 2026.xlsx").
 */
export function ExportMenu({ state, result }: ExportMenuProps) {
  const { show } = useToast();
  const [preparing, setPreparing] = useState(false);
  const chosen = useRef<Format | null>(null);

  async function exportAs(format: Format) {
    if (result && result.rows.length === 0) {
      show({ message: 'Nothing to export — clear a filter first.' });
      return;
    }
    setPreparing(true);
    try {
      const fileName = await downloadFromApi(
        api.GET('/api/all-work/export', {
          params: {
            query: { ...toQueryParams(state), format, unassigned: state.showUnassigned },
          },
          parseAs: 'blob',
        }),
        `MAPP All work.${format}`,
      );
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
          <DropdownMenu.Item
            className={styles.item}
            disabled={preparing}
            onSelect={() => {
              chosen.current = 'xlsx';
            }}
          >
            Spreadsheet (.xlsx)
          </DropdownMenu.Item>
          <DropdownMenu.Item
            className={styles.item}
            disabled={preparing}
            onSelect={() => {
              chosen.current = 'csv';
            }}
          >
            CSV (.csv)
          </DropdownMenu.Item>
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  );
}
