import type { ComponentProps, MouseEvent } from 'react';
import { Link } from 'react-router';
import type { AllWorkRow } from './all-work-types';
import { rowHref } from './row-href';

type RecordLinkProps = Omit<ComponentProps<typeof Link>, 'to' | 'onClick' | 'draggable'> & {
  row: AllWorkRow;
  onOpen: (row: AllWorkRow) => void;
};

/**
 * A link to a row's record that first checks it still exists (onOpen); never draggable, so a bar
 * cannot be dragged to other dates (FR-027).
 */
export function RecordLink({ row, onOpen, ...linkProps }: RecordLinkProps) {
  function open(event: MouseEvent<HTMLAnchorElement>) {
    if (event.button !== 0 || event.ctrlKey || event.metaKey || event.shiftKey || event.altKey) {
      return;
    }
    event.preventDefault();
    onOpen(row);
  }
  return (
    // The other props include those of a Tooltip around the link (its ref and handlers).
    <Link {...linkProps} to={rowHref(row)} draggable={false} onClick={open} />
  );
}
