import { Download, Send } from 'lucide-react';
import { useId, useState } from 'react';
import { asApiError } from '@/api/errors';
import { isShown } from '@/session/access';
import { Button } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { useFormatters } from '@/format/useFormatters';
import { useHidesUnavailable } from '@/ui/ViewOnlyNote/view-only-context';
import { downloadPurchaseRequestForm } from './api';
import type { FormFormat, PurchaseRequest, PurchaseRequestRouting } from './api';
import {
  downloadLabel,
  RESEND_FORM,
  ROUTING_SUBTITLE,
  ROUTING_TITLE,
  routedByText,
  routedText,
} from './page-text';
import { NO_FORM_YET, ROUTING_TYPE_WORDS } from './purchase-request-text';
import { ResendFormDialog } from './ResendFormDialog';
import styles from './PurchaseRequestPage.module.css';

/**
 * The routing history (032 FR-030 – FR-035): every hand-off to the requisitioner, newest first,
 * each with its PR form as routed (XLSX or CSV), and Resend PR form.
 */
export function RoutingHistory({ purchaseRequest }: { purchaseRequest: PurchaseRequest }) {
  const [resending, setResending] = useState(false);
  const [problem, setProblem] = useState<string | null>(null);
  const reasonId = useId();
  const { resend } = purchaseRequest.access;
  const resendBlocked = !resend.available;
  const hidesUnavailable = useHidesUnavailable();

  async function download(routing: PurchaseRequestRouting, format: FormFormat) {
    setProblem(null);
    try {
      await downloadPurchaseRequestForm(purchaseRequest.id, routing.id, format);
    } catch (error) {
      setProblem(asApiError(error).message);
    }
  }

  const actions = isShown(resend) &&
    !(hidesUnavailable && resendBlocked) &&
    purchaseRequest.routings.length > 0 && (
      <span className={styles.withReason}>
        <Button
          size="sm"
          icon={<Send />}
          disabled={resendBlocked}
          aria-describedby={resendBlocked ? reasonId : undefined}
          onClick={() => setResending(true)}
        >
          {RESEND_FORM}
        </Button>
        {resendBlocked && (
          <span id={reasonId} className={styles.reason}>
            {resend.reason}
          </span>
        )}
      </span>
    );

  return (
    <Card title={ROUTING_TITLE} subtitle={ROUTING_SUBTITLE} actions={actions || undefined}>
      <div className={styles.stack}>
        {problem && (
          <p role="alert" className={styles.problem}>
            {problem}
          </p>
        )}
        {purchaseRequest.routings.length === 0 ? (
          <p className={styles.note}>{NO_FORM_YET}</p>
        ) : (
          <ol role="list" className={styles.list} aria-label={ROUTING_TITLE}>
            {purchaseRequest.routings.map((routing) => (
              <RoutingRow key={routing.id} routing={routing} onDownload={download} />
            ))}
          </ol>
        )}
      </div>
      <ResendFormDialog
        purchaseRequest={purchaseRequest}
        open={resending}
        onOpenChange={setResending}
      />
    </Card>
  );
}

function RoutingRow({
  routing,
  onDownload,
}: {
  routing: PurchaseRequestRouting;
  onDownload: (routing: PurchaseRequestRouting, format: FormFormat) => Promise<void>;
}) {
  const { dateTime } = useFormatters();
  return (
    <li className={styles.item}>
      <span className={styles.itemText}>
        <span className={styles.itemTitle}>
          {routedText(ROUTING_TYPE_WORDS[routing.type], routing.recipients)}
        </span>
        <span className={styles.note}>
          {routedByText(dateTime(routing.routedAt), routing.routedBy.name, routing.lineCount)}
        </span>
      </span>
      <span className={styles.buttons}>
        <Button
          size="sm"
          icon={<Download />}
          aria-label={downloadLabel(routing.id, 'XLSX')}
          onClick={() => void onDownload(routing, 'xlsx')}
        >
          XLSX
        </Button>
        <Button
          size="sm"
          icon={<Download />}
          aria-label={downloadLabel(routing.id, 'CSV')}
          onClick={() => void onDownload(routing, 'csv')}
        >
          CSV
        </Button>
      </span>
    </li>
  );
}
