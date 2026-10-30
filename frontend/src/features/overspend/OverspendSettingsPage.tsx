import { Lock } from 'lucide-react';
import { useId, useState } from 'react';
import { asApiError, conflictMessage, isSessionLost } from '@/api/errors';
import { Button } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { useToast } from '@/ui/Toast/useToast';
import { usePageTitle } from '@/ui/usePageTitle';
import { localDate } from '@/format/intl';
import { useFormatters } from '@/format/useFormatters';
import { useOverspendConfiguration, useSaveOverspendConfiguration } from './api';
import type { OverspendApprovalMode, OverspendConfiguration, OverspendMode } from './api';
import {
  APPROVAL_HINTS,
  APPROVAL_LABEL,
  APPROVAL_MODES,
  APPROVAL_WORDS,
  DEFAULT_LABEL,
  EXISTING_KEEP,
  MODE_HINTS,
  MODE_WORDS,
  MODES,
  NEVER_CHANGED,
  OVERSPEND_CONTROL,
  SAVE,
  SETTING_SAVED,
  SETTINGS_DESCRIPTION,
  lastChangedText,
} from './overspend-text';
import styles from './OverspendSettingsPage.module.css';

/**
 * Settings › Overspend control: the setting new budgets start
 * with, and whether approving while an activity's line items exceed its planned cost is warned or
 * refused.
 */
export function OverspendSettingsPage() {
  usePageTitle(OVERSPEND_CONTROL);
  const config = useOverspendConfiguration();
  return (
    <>
      <PageHeader
        breadcrumb={[{ label: 'Settings', to: '/settings' }, { label: OVERSPEND_CONTROL }]}
        title={OVERSPEND_CONTROL}
        description={SETTINGS_DESCRIPTION}
      />
      <Card>
        {config.data ? (
          <SettingForm config={config.data} />
        ) : config.isError ? (
          <ErrorState error={config.error} onRetry={() => void config.refetch()} />
        ) : (
          <Skeleton lines={3} />
        )}
      </Card>
    </>
  );
}

function SettingForm({ config }: { config: OverspendConfiguration }) {
  const save = useSaveOverspendConfiguration();
  const { show } = useToast();
  const { date, dateFormat } = useFormatters();
  const [mode, setMode] = useState<OverspendMode>(config.defaultMode);
  const [approval, setApproval] = useState<OverspendApprovalMode>(config.approvalOverPlannedCost);
  const [problem, setProblem] = useState('');
  const locked = !config.canChange.available;

  async function send() {
    setProblem('');
    try {
      await save.mutateAsync({
        version: config.version,
        defaultMode: mode,
        approvalOverPlannedCost: approval,
      });
      show({ message: SETTING_SAVED });
    } catch (error) {
      const failure = asApiError(error);
      if (!isSessionLost(failure)) {
        setProblem(conflictMessage(failure, 'setting', dateFormat));
      }
    }
  }

  return (
    <form
      noValidate
      className={styles.stack}
      aria-busy={save.isPending}
      onSubmit={(event) => {
        event.preventDefault();
        void send();
      }}
    >
      {problem && (
        <p role="alert" className={styles.problem}>
          {problem}
        </p>
      )}
      <Choices
        name="defaultMode"
        legend={DEFAULT_LABEL}
        choices={MODES}
        words={MODE_WORDS}
        hints={MODE_HINTS}
        value={mode}
        disabled={locked}
        onChange={setMode}
        note={EXISTING_KEEP}
      />
      <Choices
        name="approvalOverPlannedCost"
        legend={APPROVAL_LABEL}
        choices={APPROVAL_MODES}
        words={APPROVAL_WORDS}
        hints={APPROVAL_HINTS}
        value={approval}
        disabled={locked}
        onChange={setApproval}
      />
      {locked && (
        <p className={styles.lock} role="note">
          <Lock aria-hidden="true" />
          <span>{config.canChange.reason}</span>
        </p>
      )}
      <p className={styles.note}>
        {config.updatedBy && config.updatedAt
          ? lastChangedText(config.updatedBy.name, date(localDate(config.updatedAt)))
          : NEVER_CHANGED}
      </p>
      {!locked && (
        <div className={styles.actions}>
          <Button type="submit" variant="primary" aria-busy={save.isPending || undefined}>
            {SAVE}
          </Button>
        </div>
      )}
    </form>
  );
}

interface ChoicesProps<T extends string> {
  name: string;
  legend: string;
  choices: readonly T[];
  words: Record<T, string>;
  hints: Record<T, string>;
  value: T;
  disabled: boolean;
  onChange: (value: T) => void;
  note?: string;
}

function Choices<T extends string>(props: ChoicesProps<T>) {
  const noteId = useId();
  return (
    <fieldset className={styles.group} aria-describedby={props.note ? noteId : undefined}>
      <legend className={styles.legend}>{props.legend}</legend>
      {props.choices.map((choice) => (
        <label key={choice} className={styles.radio}>
          <input
            type="radio"
            name={props.name}
            value={choice}
            checked={props.value === choice}
            disabled={props.disabled}
            onChange={() => props.onChange(choice)}
          />
          <span>
            <span className={styles.word}>{props.words[choice]}</span>
            <span className={styles.note}>{props.hints[choice]}</span>
          </span>
        </label>
      ))}
      {props.note && (
        <p id={noteId} className={styles.note}>
          {props.note}
        </p>
      )}
    </fieldset>
  );
}
