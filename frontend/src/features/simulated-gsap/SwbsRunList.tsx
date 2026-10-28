import { useFormatters } from '@/format/useFormatters';
import type { SwbsRun, SwbsRunCode } from './api';
import { NO_RUNS, NOTHING_SENT, ranByText, shownText, SOURCE_TEXT } from './simulated-gsap-text';
import styles from './SimulatedGsapPanel.module.css';

export function SwbsRunList({ runs, total }: { runs: SwbsRun[]; total: number }) {
  const { dateTime } = useFormatters();
  if (runs.length === 0) {
    return <p className={styles.muted}>{NO_RUNS}</p>;
  }
  return (
    <>
      <ol className={styles.list} aria-label="Runs, newest first">
        {runs.map((run) => (
          <li key={run.id} className={styles.item}>
            <p className={styles.name}>{`${SOURCE_TEXT[run.source]} · ${run.id}`}</p>
            <p className={styles.muted}>{ranByText(dateTime(run.ranAt), run.ranBy.name)}</p>
            <CodeList codes={run.codes} />
          </li>
        ))}
      </ol>
      {total > runs.length && <p className={styles.muted}>{shownText(runs.length, total)}</p>}
    </>
  );
}

export function CodeList({ codes }: { codes: SwbsRunCode[] }) {
  if (codes.length === 0) {
    return <p>{NOTHING_SENT}</p>;
  }
  return (
    <ul className={styles.codes}>
      {codes.map((code) => (
        <li key={code.lineItemId}>
          {`${code.lineItemName} (${code.lineItemId}) · `}
          <span className={styles.code}>{code.swbsCode}</span>
        </li>
      ))}
    </ul>
  );
}
