import { useEffect, useRef, useState } from 'react';

/** What useCloseAfterSave gives a form panel. */
export interface CloseAfterSave {
  /** The save went through: the form no longer counts as changed (pass `isDirty && !saved`). */
  saved: boolean;
  /** Call once the save went through; the panel closes on the next render. */
  closeAfterSave: () => void;
}

/**
 * Closes a form panel whose being open is kept in the address (`?key=`, `?person=`) once its save
 * has gone through. Closing changes the address, which the unsaved-changes guard
 * (useUnsavedChanges) would stop with "Discard changes?" while the form still counts as changed;
 * so the form first stops counting as changed, and closes on the next render, once the guard
 * knows.
 *
 *   const { saved, closeAfterSave } = useCloseAfterSave(onClose);
 *   <FormPanel isDirty={!saved && text !== start} … />
 *   // after a successful save:
 *   closeAfterSave();
 */
export function useCloseAfterSave(onClose: () => void): CloseAfterSave {
  const [saved, setSaved] = useState(false);
  const close = useRef(onClose);

  useEffect(() => {
    close.current = onClose;
  });

  useEffect(() => {
    if (saved) {
      close.current();
    }
  }, [saved]);

  return { saved, closeAfterSave: () => setSaved(true) };
}
