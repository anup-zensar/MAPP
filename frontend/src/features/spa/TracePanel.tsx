import { Link } from 'react-router';
import { useFormatters } from '@/format/useFormatters';
import { Button } from '@/ui/Button/Button';
import { SidePanel } from '@/ui/SidePanel/SidePanel';
import { recordHref } from '@/ui/record-links';
import type { TraceRequest } from './FigureButton';
import { TRACE } from './spa-text';
import styles from './Spa.module.css';

interface TracePanelProps {
  planId: string;
  request: TraceRequest | null;
  onClose: () => void;
}

/** The activities behind a figure, with their amounts and the figure as their total (013 FR-007). */
export function TracePanel({ planId, request, onClose }: TracePanelProps) {
  const { money } = useFormatters();
  return (
    <SidePanel
      open={request !== null}
      onOpenChange={(open) => !open && onClose()}
      title={request ? TRACE.title(request.figure) : ''}
      footer={<Button onClick={onClose}>{TRACE.close}</Button>}
    >
      {request && request.items.length === 0 && <p className={styles.note}>{TRACE.none}</p>}
      {request && request.items.length > 0 && (
        <table className={styles.table}>
          <caption className="visually-hidden">{TRACE.title(request.figure)}</caption>
          <thead>
            <tr>
              <th scope="col">Activity</th>
              <th scope="col" className={styles.numeric}>
                Amount
              </th>
            </tr>
          </thead>
          <tbody>
            {request.items.map((item) => (
              <tr key={item.activityId}>
                <th scope="row">
                  <Link to={recordHref('activity', item.activityId, planId)}>{item.name}</Link>
                </th>
                <td className={styles.numeric}>{money(item.amount)}</td>
              </tr>
            ))}
          </tbody>
          <tfoot>
            <tr className={styles.totals}>
              <th scope="row">{TRACE.total}</th>
              <td className={styles.numeric}>{money(request.total)}</td>
            </tr>
          </tfoot>
        </table>
      )}
    </SidePanel>
  );
}
