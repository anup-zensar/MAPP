import { useState } from 'react';
import { CheckboxField } from '@/ui/CheckboxField/CheckboxField';
import { ChoiceField } from '@/ui/ChoiceField/ChoiceField';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { useBulkFields } from './api';
import { BulkDialog, Problem } from './BulkDialog';
import styles from './bulk.module.css';
import { NOT_OFFERED_FIELDS, recordsText } from './bulk-text';
import type { BulkField, BulkFieldValue, BulkRequest, BulkResult } from './bulk-types';
import type { BulkActionDialogProps } from './ChangeOwnerDialog';
import { FieldValueInput } from './FieldValueInput';
import type { ValueDraft } from './FieldValueInput';

const EMPTY: ValueDraft = { date: '', optionIds: [], yesNo: '' };

/** Change field (FR-009 – FR-011): one value for every selected record of one level, or Clear the value. */
export function ChangeFieldDialog({ records, onClose, onDone }: BulkActionDialogProps) {
  const level = records[0]?.level ?? 'activity';
  const fields = useBulkFields(level, true);
  const [key, setKey] = useState('');
  const [draft, setDraft] = useState<ValueDraft>(EMPTY);
  const [clear, setClear] = useState(false);
  const [restart, setRestart] = useState(false);
  const field = fields.data?.items.find((f) => f.key === key);
  const value = field ? valueOf(field, draft) : null;
  const request: BulkRequest | null =
    field && (clear || value)
      ? {
          action: 'changeField',
          recordIds: records.map((r) => r.id),
          field: { key, value: clear ? null : value },
          restartApproval: restart,
        }
      : null;

  return (
    <BulkDialog
      title="Change field"
      request={request}
      submitLabel={() => 'Change field'}
      describe={(preview) => describe(preview, field, clear, restart, setRestart)}
      onClose={onClose}
      onDone={onDone}
    >
      {fields.isPending && <Skeleton lines={2} />}
      {fields.error && <Problem error={fields.error} />}
      {fields.data && (
        <>
          <ChoiceField
            label="Field"
            name="field"
            options={fields.data.items.map((f) => ({ id: f.key, label: f.label, active: true }))}
            value={key}
            onChange={(next) => {
              setKey(next);
              setDraft(EMPTY);
              setClear(false);
            }}
            note={NOT_OFFERED_FIELDS}
          />
          {field && !clear && <FieldValueInput field={field} draft={draft} onChange={setDraft} />}
          {field && !field.required && (
            <CheckboxField
              label="Clear the value"
              name="clear"
              checked={clear}
              onChange={setClear}
            />
          )}
        </>
      )}
    </BulkDialog>
  );
}

function valueOf(field: BulkField, draft: ValueDraft): BulkFieldValue | null {
  switch (field.kind) {
    case 'date':
      return draft.date ? { date: draft.date } : null;
    case 'yesNo':
      return draft.yesNo ? { yesNo: draft.yesNo === 'yes' } : null;
    default:
      return draft.optionIds.length > 0 ? { optionIds: draft.optionIds } : null;
  }
}

function describe(
  preview: BulkResult,
  field: BulkField | undefined,
  clear: boolean,
  restart: boolean,
  setRestart: (restart: boolean) => void,
) {
  const verb = clear ? 'Clear' : 'Set';
  return (
    <>
      <p>
        {verb} {field?.label} on {recordsText(preview.counts.done)}.
      </p>
      {preview.reapproval.length > 0 && (
        <fieldset className={styles.sentences}>
          <legend>
            These {preview.reapproval.length} activities are submitted or approved — changing their
            dates sends them back for approval:
          </legend>
          <ul className={styles.list}>
            {preview.reapproval.map((r) => (
              <li key={r.recordId}>
                {r.name} ({r.status})
              </li>
            ))}
          </ul>
          <label>
            <input
              type="radio"
              name="restart"
              checked={!restart}
              onChange={() => setRestart(false)}
            />{' '}
            Leave those unchanged
          </label>
          <label>
            <input
              type="radio"
              name="restart"
              checked={restart}
              onChange={() => setRestart(true)}
            />{' '}
            Change and restart approval
          </label>
        </fieldset>
      )}
    </>
  );
}
