import { useAttributeForm } from '@/features/settings';
import { Button } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { ShellAttributeValues } from '@/ui/ShellAttributeFields/ShellAttributeValues';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { localDate } from '@/format/intl';
import { useFormatters } from '@/format/useFormatters';
import type { PurchaseRequest } from './api';
import { Detail, DetailsList } from './Details';
import {
  ATTRIBUTES_FAILED,
  bookkeepingText,
  DETAILS_TITLE,
  inactive,
  SHELL_ATTRIBUTES,
  TRY_AGAIN,
} from './page-text';
import { FIELD_LABELS, FILLED_BY_GSAP } from './purchase-request-text';
import styles from './PurchaseRequestPage.module.css';

/**
 * The PR's header (032 FR-005, FR-006, FR-008): its standard fields, the values the server derives
 * from its lines (company code, cost centres, HX keys, totals) and its Shell attribute values.
 */
export function PurchaseRequestDetailsCard({
  purchaseRequest: pr,
}: {
  purchaseRequest: PurchaseRequest;
}) {
  const { date, money } = useFormatters();
  const workflow = (ref: PurchaseRequest['approvalWorkflow']) =>
    ref ? inactive(ref.name, ref.active) : null;
  return (
    <Card title={DETAILS_TITLE}>
      <div className={styles.stack}>
        <DetailsList>
          <Detail label={FIELD_LABELS.name}>{pr.name}</Detail>
          <Detail label={FIELD_LABELS.description}>
            {pr.description ? <span className={styles.description}>{pr.description}</span> : null}
          </Detail>
          <Detail label={FIELD_LABELS.owner}>
            {pr.owner ? inactive(pr.owner.name, pr.owner.active) : null}
          </Detail>
          <Detail label={FIELD_LABELS.moaApprover}>
            {pr.moaApprover ? inactive(pr.moaApprover.name, pr.moaApprover.active) : null}
          </Detail>
          <Detail label={FIELD_LABELS.approvalWorkflow}>{workflow(pr.approvalWorkflow)}</Detail>
          <Detail label={FIELD_LABELS.modificationWorkflow}>
            {workflow(pr.modificationWorkflow)}
          </Detail>
          <Detail label={FIELD_LABELS.amount} numeric>
            {money(pr.amountDisplay)}
          </Detail>
          <Detail label={FIELD_LABELS.companyCode}>{pr.companyCode}</Detail>
          <Detail label={FIELD_LABELS.costCentres}>{pr.costCentres.join(', ')}</Detail>
          <Detail label={FIELD_LABELS.hxKeys}>{pr.hxKeys.join(', ')}</Detail>
          <Detail label={FIELD_LABELS.invoiced} tip={FILLED_BY_GSAP} numeric>
            {money(pr.invoicedDisplay)}
          </Detail>
        </DetailsList>
        <Attributes purchaseRequest={pr} />
        <p className={styles.bookkeeping}>
          {bookkeepingText(
            pr.createdBy.name,
            date(localDate(pr.createdAt)),
            pr.updatedBy.name,
            date(localDate(pr.updatedAt)),
          )}
        </p>
      </div>
    </Card>
  );
}

function Attributes({ purchaseRequest }: { purchaseRequest: PurchaseRequest }) {
  const form = useAttributeForm('purchaseRequest');
  return (
    <section className={styles.stack} aria-label={SHELL_ATTRIBUTES}>
      {form.data ? (
        <ShellAttributeValues
          form={form.data}
          values={purchaseRequest.attributes}
          businessUnit={[]}
          sectionHeading="h3"
        />
      ) : form.isError ? (
        <p className={styles.withReason}>
          <span className={styles.problem}>{ATTRIBUTES_FAILED}</span>
          <Button size="sm" onClick={() => void form.refetch()}>
            {TRY_AGAIN}
          </Button>
        </p>
      ) : (
        <Skeleton lines={3} />
      )}
    </section>
  );
}
