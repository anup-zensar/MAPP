import { Link } from 'react-router';
import type { components } from '@/api/generated/schema';
import { FormDialog } from '@/ui/FormDialog/FormDialog';
import { fieldHref } from '@/ui/record-links';
import { groupLabel } from '../plan-text';
import { isPlanningType } from '../planning-type';
import styles from './dialogs.module.css';

type ReadinessGroup = components['schemas']['ReadinessGroup'];

export interface PublishBlockedDialogProps {
  planId: string;
  open: boolean;
  /** Every needed field that stops publishing, grouped by record. */
  missing: ReadinessGroup[];
  onClose: () => void;
}

/**
 * Why a plan can't be published yet (FR-024): "Publish needs <n> fields", grouped by record — the
 * plan, then each campaign or initiative — each field a link to fill it in. The plan stays Draft.
 */
export function PublishBlockedDialog({
  open,
  planId,
  missing,
  onClose,
}: PublishBlockedDialogProps) {
  const groups = missing;
  const count = groups.reduce((total, group) => total + group.fields.length, 0);
  return (
    <FormDialog
      open={open}
      onOpenChange={(open) => !open && onClose()}
      title={count === 1 ? 'Publish needs 1 field' : `Publish needs ${count} fields`}
      description="Fill these in, then publish again:"
      submitLabel="Close"
      hideCancel
      onSubmit={onClose}
    >
      <div className={styles.stack}>
        {groups.map(({ record, fields }) => (
          <section key={`${record.type}:${record.id}`} className={styles.group}>
            <h3 className={styles.groupTitle}>{groupLabel(record)}</h3>
            <ul className={styles.list}>
              {fields.map((field) => (
                <li key={field.field}>
                  {isPlanningType(record.type) ? (
                    <Link
                      to={fieldHref(record.type, record.id, field.field, planId)}
                      onClick={onClose}
                    >
                      {field.label}
                    </Link>
                  ) : (
                    field.label
                  )}
                </li>
              ))}
            </ul>
          </section>
        ))}
      </div>
    </FormDialog>
  );
}
