import { Send } from 'lucide-react';
import { useId, useState } from 'react';
import { ApiError } from '@/api/errors';
import { Button } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { useSendSwbsRequests, useSwbsRuns } from './api';
import type { SwbsRun, SwbsRunPage } from './api';
import {
  HISTORY_TITLE,
  NO_ACCESS,
  NOTHING_READY,
  PANEL_SUBTITLE,
  PANEL_TITLE,
  readyText,
  returnedText,
  SEND_ALL,
  SEND_ALL_TEXT,
  sendAllTitle,
} from './simulated-gsap-text';
import { GsapProgress } from './GsapProgress';
import { CodeList, SwbsRunList } from './SwbsRunList';
import styles from './SimulatedGsapPanel.module.css';

/** The Financials page's Simulated GSAP panel: what is ready, the send action, the runs. */
export function SimulatedGsapPanel() {
  const runs = useSwbsRuns();
  const forbidden = runs.error instanceof ApiError && runs.error.kind === 'forbidden';
  return (
    <Card title={PANEL_TITLE} subtitle={PANEL_SUBTITLE}>
      {runs.data ? (
        <Loaded page={runs.data} />
      ) : forbidden ? (
        <p className={styles.muted}>{NO_ACCESS}</p>
      ) : runs.isError ? (
        <ErrorState error={runs.error} onRetry={() => void runs.refetch()} />
      ) : (
        <Skeleton lines={3} />
      )}
    </Card>
  );
}

function Loaded({ page }: { page: SwbsRunPage }) {
  const send = useSendSwbsRequests();
  const reasonId = useId();
  const [confirming, setConfirming] = useState(false);
  const [result, setResult] = useState<SwbsRun | null>(null);
  const nothingReady = page.readyToSend === 0;

  function handleConfirm() {
    setConfirming(false);
    setResult(null);
    send.mutate(undefined, { onSuccess: (run) => setResult(run) });
  }

  return (
    <div className={styles.stack}>
      <div className={styles.action}>
        {!nothingReady && <p className={styles.ready}>{readyText(page.readyToSend)}</p>}
        <Button
          icon={<Send />}
          disabled={nothingReady || send.isPending}
          aria-describedby={nothingReady ? reasonId : undefined}
          onClick={() => setConfirming(true)}
        >
          {SEND_ALL}
        </Button>
        {nothingReady && (
          <p id={reasonId} className={styles.muted}>
            {NOTHING_READY}
          </p>
        )}
      </div>
      {send.isPending && <GsapProgress />}
      {result && (
        <div role="status" className={styles.result}>
          <p className={styles.name}>{returnedText(result.sent)}</p>
          <CodeList codes={result.codes} />
        </div>
      )}
      {send.error && (
        <p role="alert" className={styles.problem}>
          {send.error.message}
        </p>
      )}
      <section aria-labelledby={`${reasonId}-history`} className={styles.stack}>
        <h3 id={`${reasonId}-history`} className={styles.heading}>
          {HISTORY_TITLE}
        </h3>
        <SwbsRunList runs={page.items} total={page.total} />
      </section>
      <ConfirmDialog
        open={confirming}
        onOpenChange={setConfirming}
        title={sendAllTitle(page.readyToSend)}
        confirmLabel={SEND_ALL}
        onConfirm={handleConfirm}
        busy={send.isPending}
      >
        <p>{SEND_ALL_TEXT}</p>
      </ConfirmDialog>
    </div>
  );
}
