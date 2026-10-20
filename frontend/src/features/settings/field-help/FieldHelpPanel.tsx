import { RotateCcw } from 'lucide-react';
import { useState } from 'react';
import { ApiError, conflictMessage } from '@/api/errors';
import { Button, FormPanel, problemOf, TextAreaField, useToast } from '@/components';
import type { FieldProblem, SaveFailure } from '@/components';
import { useFormatters } from '@/lib/useFormatters';
import { CONFLICT_NEXT_STEP } from '../shared/saveFailure';
import { fetchFieldHelp, HELP_MAX_LENGTH, useUpdateFieldHelp } from './fieldHelpApi';
import type { FieldHelpEntry } from './fieldHelpApi';
import { entryName, TOO_LONG } from './fieldHelpWords';
import { ResetHelpDialog } from './ResetHelpDialog';
import styles from './fieldHelp.module.css';

export interface FieldHelpPanelProps {
  entry: FieldHelpEntry;
  onClose: () => void;
}

/**
 * Edit field help (016 FR-046) in a side panel: the text (plain, 0–300 characters, counting down
 * near the limit; empty hides the ⓘ), the default beside it and **Reset to default** with a
 * question first. A save over a newer change is refused, naming who and when.
 */
export function FieldHelpPanel({ entry, onClose }: FieldHelpPanelProps) {
  const { show } = useToast();
  const { dateFormat, dateTime } = useFormatters();
  const update = useUpdateFieldHelp();
  const [start] = useState(entry.text);
  const [text, setText] = useState(entry.text);
  const [version, setVersion] = useState(entry.version);
  const [saving, setSaving] = useState(false);
  const [problems, setProblems] = useState<FieldProblem[]>([]);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  const [askReset, setAskReset] = useState(false);
  const name = `'${entry.label}'`;

  async function save() {
    // Line breaks are not kept: the tip is one paragraph (data-model §2.5).
    const typed = text.replace(/\s*\n\s*/g, ' ').trim();
    if ([...typed].length > HELP_MAX_LENGTH) {
      setProblems([{ field: 'text', message: TOO_LONG }]);
      setFailure(null);
      return;
    }
    if (typed === start.trim()) {
      onClose();
      return;
    }
    setSaving(true);
    try {
      await update.mutateAsync({ key: entry.key, body: { text: typed, version } });
      show({ message: `Help of ${name} saved` });
      onClose();
    } catch (error) {
      await showFailure(error);
    } finally {
      setSaving(false);
    }
  }

  async function showFailure(error: unknown) {
    const apiError = error instanceof ApiError ? error : new ApiError('other');
    if (apiError.kind === 'signInRequired' || apiError.kind === 'sessionExpired') {
      return;
    }
    if (apiError.kind === 'validation') {
      setProblems(apiError.fields);
      setFailure(null);
      return;
    }
    setProblems([]);
    if (apiError.kind === 'conflict') {
      setFailure({
        message: conflictMessage(apiError, 'help text', dateFormat),
        nextStep: CONFLICT_NEXT_STEP,
      });
      const latest = await fetchFieldHelp(entry.key).catch(() => undefined);
      if (latest) {
        setVersion(latest.version);
      }
      return;
    }
    setFailure({ message: apiError.message });
  }

  return (
    <>
      <FormPanel
        title="Edit field help"
        description={entryName(entry.level, entry.label)}
        isDirty={text !== start}
        saving={saving}
        problems={problems}
        failure={failure}
        onSave={() => void save()}
        onClose={onClose}
      >
        <div className={styles.form}>
          <TextAreaField
            label="Help text"
            name="text"
            hint="Plain text, shown in the ⓘ beside the field. Leave it empty to hide the ⓘ."
            value={text}
            onChange={setText}
            maxLength={HELP_MAX_LENGTH}
            rows={5}
            error={problemOf(problems, 'text')}
          />
          <div className={styles.defaultText}>
            <h3>Default text</h3>
            <p>{entry.defaultText}</p>
            {entry.isDefault ? (
              <p className={styles.meta}>This field uses the default text.</p>
            ) : (
              <>
                {entry.updatedBy && entry.updatedAt && (
                  <p className={styles.meta}>
                    {`Edited by ${entry.updatedBy.name}, ${dateTime(entry.updatedAt)}`}
                  </p>
                )}
                <div>
                  <Button size="sm" icon={<RotateCcw />} onClick={() => setAskReset(true)}>
                    Reset to default
                  </Button>
                </div>
              </>
            )}
          </div>
        </div>
      </FormPanel>
      {askReset && (
        <ResetHelpDialog
          entry={entry}
          version={version}
          onDone={onClose}
          onCancel={() => setAskReset(false)}
        />
      )}
    </>
  );
}
