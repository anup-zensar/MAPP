/**
 * A ref callback for a dialog's fields that arrive after it opened (the move and copy dialogs load
 * their destinations first): the first field that can be used takes the focus, which FormDialog
 * had put on the submit button meanwhile — unless the person already moved it into a field.
 */
export function focusOnArrival(node: HTMLElement | null): void {
  if (!node || document.activeElement instanceof HTMLInputElement) {
    return;
  }
  node.querySelector<HTMLInputElement>('input:not([disabled])')?.focus();
}
