import { Dialog } from 'radix-ui';
import { Button } from '@/ui/Button/Button';
import type { PurchaseRequestLineItemOption } from './api';
import { ADD_LINE, PICKER } from './form-text';
import { LineItemOptionsList } from './LineItemOptionsList';
import type { LineItemOptionsListProps } from './LineItemOptionsList';
import styles from './LineItemPicker.module.css';

export interface LineItemPickerProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  query: LineItemOptionsListProps['query'];
  inForm: string[];
  onChoose: (option: PurchaseRequestLineItemOption) => void;
}

/** Add line (032 FR-010): the line-item search in a dialog; Escape or Close leaves it. */
export function LineItemPicker({
  open,
  onOpenChange,
  query,
  inForm,
  onChoose,
}: LineItemPickerProps) {
  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className={styles.overlay} />
        <Dialog.Content className={styles.dialog}>
          <Dialog.Title className={styles.title}>{PICKER.title}</Dialog.Title>
          <Dialog.Description className={styles.description}>
            {PICKER.description}
          </Dialog.Description>
          {open && (
            <LineItemOptionsList
              query={query}
              inForm={inForm}
              action={ADD_LINE}
              onChoose={(option) => {
                onChoose(option);
                onOpenChange(false);
              }}
            />
          )}
          <div className={styles.actions}>
            <Dialog.Close asChild>
              <Button>{PICKER.close}</Button>
            </Dialog.Close>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
