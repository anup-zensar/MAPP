import type { ReactNode } from 'react';
import { useFormatters } from '@/format/useFormatters';
import { InfoTip } from '@/ui/InfoTip/InfoTip';
import { ProgressBar } from '@/ui/ProgressBar/ProgressBar';
import type { SpaView } from './api';
import { FigureButton } from './FigureButton';
import type { TraceRequest } from './FigureButton';
import { HEADLINE, OVERSPEND, percentText } from './spa-text';
import styles from './Spa.module.css';

interface TileProps {
  label: string;
  tip: string;
  children: ReactNode;
  detail?: ReactNode;
}

function Tile({ label, tip, children, detail }: TileProps) {
  return (
    <div className={styles.tile}>
      <p className={styles.label}>
        {label}
        <InfoTip label={label} text={tip} />
      </p>
      <div className={styles.value}>{children}</div>
      {detail}
    </div>
  );
}

function Overspend({ when }: { when: boolean }) {
  return when ? <p className={styles.problemText}>{OVERSPEND}</p> : null;
}

/** The headline figures (013 FR-015, FR-016): overspend in red and in words. */
export function SpaHeadline({
  view,
  onTrace,
}: {
  view: SpaView;
  onTrace: (request: TraceRequest) => void;
}) {
  const { money } = useFormatters();
  const h = view.headline;
  const utilisation =
    h.utilisationPercent === null ? null : Math.round(Number(h.utilisationPercent));
  return (
    <section className={styles.tiles} aria-label="Headline figures">
      <Tile {...HEADLINE.plannedCost}>
        <FigureButton
          figure={HEADLINE.plannedCost.label}
          value={h.plannedCost}
          trace={h.plannedCostTrace}
          onTrace={onTrace}
        />
      </Tile>
      <Tile {...HEADLINE.committed}>
        <FigureButton
          figure={HEADLINE.committed.label}
          value={h.committed}
          trace={h.committedTrace}
          onTrace={onTrace}
        />
      </Tile>
      <Tile {...HEADLINE.actualSpend}>
        <FigureButton
          figure={HEADLINE.actualSpend.label}
          value={h.actualSpend}
          trace={h.actualSpendTrace}
          onTrace={onTrace}
        />
      </Tile>
      <Tile {...HEADLINE.remaining} detail={<Overspend when={h.overspend} />}>
        <span className={h.overspend ? styles.problem : undefined}>{money(h.remaining)}</span>
      </Tile>
      <Tile
        {...HEADLINE.utilisation}
        detail={
          utilisation === null ? (
            <p className={styles.detail}>{h.noPlannedCostReason ?? view.financeNote}</p>
          ) : (
            <ProgressBar
              value={utilisation}
              label={HEADLINE.utilisation.label}
              tone={h.overspend ? 'problem' : undefined}
            />
          )
        }
      >
        <span className={h.overspend ? styles.problem : undefined}>
          {percentText(h.utilisationPercent)}
        </span>
      </Tile>
      <Tile {...HEADLINE.variance} detail={<Overspend when={h.overspend} />}>
        <span className={h.overspend ? styles.problem : undefined}>
          {percentText(h.variancePercent, true)}
        </span>
      </Tile>
    </section>
  );
}
