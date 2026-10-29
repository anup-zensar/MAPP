import { Download, LoaderCircle } from 'lucide-react';
import { DropdownMenu } from 'radix-ui';
import { useRef, useState } from 'react';
import { ApiError } from '@/api/errors';
import { Button } from '@/ui/Button/Button';
import { useToast } from '@/ui/Toast/useToast';
import type { ExportFormat } from './api';
import styles from './Dashboards.module.css';

const FORMATS: { format: ExportFormat; label: string }[] = [
  { format: 'pptx', label: 'PowerPoint (.pptx)' },
  { format: 'pdf', label: 'PDF (.pdf)' },
];

interface ExportMenuProps {
  download: (format: ExportFormat) => Promise<string>;
}

export function ExportMenu({ download }: ExportMenuProps) {
  const { show } = useToast();
  const [preparing, setPreparing] = useState(false);
  const chosen = useRef<ExportFormat | null>(null);

  async function exportAs(format: ExportFormat) {
    setPreparing(true);
    try {
      const fileName = await download(format);
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
          icon={preparing ? <LoaderCircle /> : <Download />}
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
          {FORMATS.map((item) => (
            <DropdownMenu.Item
              key={item.format}
              className={styles.item}
              disabled={preparing}
              onSelect={() => {
                chosen.current = item.format;
              }}
            >
              {item.label}
            </DropdownMenu.Item>
          ))}
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  );
}
