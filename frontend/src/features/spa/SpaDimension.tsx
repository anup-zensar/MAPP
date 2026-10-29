import { useId } from 'react';
import { useFormatters } from '@/format/useFormatters';
import { Card } from '@/ui/Card/Card';
import type { SpaView } from './api';
import { Bar } from './Bar';
import { FigureButton } from './FigureButton';
import type { TraceRequest } from './FigureButton';
import { DIMENSION } from './spa-text';
import styles from './Spa.module.css';

interface SpaDimensionProps {
  view: SpaView;
  onDimension: (key: string) => void;
  onTrace: (request: TraceRequest) => void;
}

/** Planned cost by the dimension the person chooses, largest first (013 FR-018). */
export function SpaDimension({ view, onDimension, onTrace }: SpaDimensionProps) {
  const id = useId();
  const { money } = useFormatters();
  const breakdown = view.dimension;
  return (
    <Card
      title={DIMENSION.title}
      actions={
        <label className={styles.picker} htmlFor={id}>
          {DIMENSION.choose}
          <select
            id={id}
            className={styles.select}
            value={breakdown.key}
            onChange={(event) => onDimension(event.target.value)}
          >
            {view.dimensions.map((dimension) => (
              <option key={dimension.key} value={dimension.key}>
                {dimension.label}
              </option>
            ))}
          </select>
        </label>
      }
    >
      {breakdown.countsInEach && <p className={styles.note}>{DIMENSION.countsInEach}</p>}
      <table className={styles.table}>
        <caption className="visually-hidden">Planned cost by {breakdown.label}</caption>
        <thead>
          <tr>
            <th scope="col">{breakdown.label}</th>
            <th scope="col" className={styles.numeric}>
              Planned cost
            </th>
            <th scope="col">{DIMENSION.share}</th>
          </tr>
        </thead>
        <tbody>
          {breakdown.rows.map((row) => (
            <tr key={row.key}>
              <th scope="row">{row.label}</th>
              <td className={styles.numeric}>
                <FigureButton
                  figure={`${breakdown.label}: ${row.label}`}
                  value={row.amount}
                  trace={row.trace}
                  onTrace={onTrace}
                />
              </td>
              <td>
                <div className={styles.barCell}>
                  <span className={styles.shareText}>{row.share}%</span>
                  <Bar length={row.bar} series="share" />
                </div>
              </td>
            </tr>
          ))}
        </tbody>
        <tfoot>
          <tr className={styles.totals}>
            <th scope="row">Total</th>
            <td className={styles.numeric}>{money(breakdown.total)}</td>
            <td />
          </tr>
        </tfoot>
      </table>
    </Card>
  );
}
