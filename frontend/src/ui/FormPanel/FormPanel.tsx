import { CircleAlert } from 'lucide-react';
import { useId } from 'react';
import type { FormEvent, ReactNode } from 'react';
import { useUnsavedChanges } from '@/lib/useUnsavedChanges';
import { Button } from '../Button/Button';
import { FormErrorSummary } from '../FormErrorSummary/FormErrorSummary';
import type { FieldProblem } from '../FormErrorSummary/FormErrorSummary';
import { SidePanel } from '../SidePanel/SidePanel';
import styles from './FormPanel.module.css';
import type { SaveFailure } from './problemOf';

export interface FormPanelProps {
  title: string;
  description?: string;
  /** Something was typed that is not saved yet. */
  isDirty: boolean;
  /** A save is on its way; another Save is ignored meanwhile. */
  saving: boolean;
  /** The field problems of the last save, all at once. */
  problems: FieldProblem[];
  failure: SaveFailure | null;
  onSave: () => void;
  onClose: () => void;
  /** The primary button's label ("Save" when left out). */
  saveLabel?: string;
  /** The form's fields. */
  children: ReactNode;
}

/**
 * A side-panel form (design system §6.2, moved here from feature 001's developer page for 002, 003,
 * 005 and 008): Save and Cancel at the bottom, a failure and the error summary above the fields,
 * and the "Discard changes?" question before typed changes are lost — on Cancel, Close, Escape, a
 * click beside the panel, or when leaving the page. Save is never disabled.
 */
export function FormPanel({
  title,
  description,
  isDirty,
  saving,
  problems,
  failure,
  onSave,
  onClose,
  saveLabel = 'Save',
  children,
}: FormPanelProps) {
  const { dialog, confirmDiscard } = useUnsavedChanges(isDirty);
  // Save sits in the panel's footer, outside the form; the form attribute connects them.
  const formId = useId();

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    // A second Save while saving is ignored; nothing is greyed out.
    if (!saving) {
      onSave();
    }
  }

  return (
    <>
      <SidePanel
        open
        onOpenChange={(open) => {
          if (!open) {
            confirmDiscard(onClose);
          }
        }}
        title={title}
        description={description}
        footer={
          <>
            <Button type="submit" form={formId} variant="primary">
              {saveLabel}
            </Button>
            <Button onClick={() => confirmDiscard(onClose)}>Cancel</Button>
          </>
        }
      >
        <form id={formId} noValidate aria-busy={saving} className={styles.form} onSubmit={submit}>
          {failure && <FailureMessage failure={failure} />}
          <FormErrorSummary errors={problems} />
          {children}
        </form>
      </SidePanel>
      {dialog}
    </>
  );
}

/** A failure that is not about one field, such as a conflict; screen readers hear it at once. */
export function FailureMessage({ failure }: { failure: SaveFailure }) {
  return (
    <div role="alert" className={styles.failure}>
      <CircleAlert />
      <div>
        <p className={styles.failureMessage}>{failure.message}</p>
        {failure.nextStep && <p>{failure.nextStep}</p>}
      </div>
    </div>
  );
}
