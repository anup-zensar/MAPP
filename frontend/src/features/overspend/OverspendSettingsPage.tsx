import { Lock } from 'lucide-react';
import { useState } from 'react';
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
import type { OverspendConfiguration, OverspendMode } from './api';
import {
  DEFAULT_LABEL,
  EXISTING_KEEP,
  MODE_HINTS,
  MODE_WORDS,
  MODES,
  OVERSPEND_CONTROL,
  SAVE,
  SETTING_SAVED,
  SETTINGS_DESCRIPTION,
  lastChangedText,
} from './overspend-text';
import styles from './OverspendSettingsPage.module.css';

/** Settings › Overspend control (037 FR-006, FR-007): the setting new budgets start with. */
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
  const [problem, setProblem] = useState('');
  const locked = !config.canChange.available;

  async function send() {
    setProblem('');
    try {
      await save.mutateAsync({ version: config.version, defaultMode: mode });
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
      <fieldset className={styles.group} aria-describedby="overspend-default-note">
        <legend className={styles.legend}>{DEFAULT_LABEL}</legend>
        {MODES.map((choice) => (
          <label key={choice} className={styles.radio}>
            <input
              type="radio"
              name="defaultMode"
              value={choice}
              checked={mode === choice}
              disabled={locked}
              onChange={() => setMode(choice)}
            />
            <span>
              <span className={styles.word}>{MODE_WORDS[choice]}</span>
              <span className={styles.note}>{MODE_HINTS[choice]}</span>
            </span>
          </label>
        ))}
        <p id="overspend-default-note" className={styles.note}>
          {EXISTING_KEEP}
        </p>
      </fieldset>
      {locked && (
        <p className={styles.lock} role="note">
          <Lock aria-hidden="true" />
          <span>{config.canChange.reason}</span>
        </p>
      )}
      {config.updatedBy && config.updatedAt && (
        <p className={styles.note}>
          {lastChangedText(config.updatedBy.name, date(localDate(config.updatedAt)))}
        </p>
      )}
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
