import { ChevronRight, Info } from 'lucide-react';
import { Fragment, useState } from 'react';
import type { ReactNode } from 'react';
import { Link, useLocation, useNavigate, useParams } from 'react-router';
import { ApiError, conflictMessage } from '@/api/errors';
import {
  Button,
  ConfirmDialog,
  ErrorState,
  SidePanel,
  Skeleton,
  StatusChip,
  useToast,
} from '@/components';
import { localDate } from '@/lib/format';
import { recordHref } from '@/lib/recordLinks';
import { useFormatters } from '@/lib/useFormatters';
import { useFinancialTransaction, useUnlinkTransaction } from './financialsApi';
import type { FinancialRecordRef, FinancialTransaction } from './financialsApi';
import {
  currencyMissingText,
  LINE_ITEM_IN_BIN,
  matchedByText,
  monthText,
  NO_ACCESS,
  NOT_LINKED,
  recordLabel,
  ruleLinkedNote,
  TRANSACTION_GONE,
  TYPE_WORDS,
  unlinkQuestion,
} from './financialsText';
import { LinkTransactionDialog } from './LinkTransactionDialog';
import styles from './TransactionPanel.module.css';

/**
 * A transaction's details panel (015 US3 scenarios 3–4, FR-022, FR-026), at
 * /financials/transactions/<id> over the Financials page: every field as GSAP sent it, its USD and
 * display amounts, the linked path with a link per level, how it was matched or why it is not
 * linked, the import it came from, and when it was added and last updated. A not-linked
 * transaction offers Link to a line item; a manual link offers Unlink; a link made by GSAP data
 * says where to correct it.
 */
export function TransactionPanel() {
  const { transactionId = '' } = useParams();
  const location = useLocation();
  const navigate = useNavigate();
  const transaction = useFinancialTransaction(transactionId);
  const [dialog, setDialog] = useState<'link' | 'unlink' | null>(null);
  const data = transaction.data;
  const failure = transaction.error instanceof ApiError ? transaction.error : null;

  const close = () => void navigate(`/financials${location.search}`);

  const footer = data ? (
    <>
      {data.link.state === 'notLinked' && (
        <Button variant="primary" onClick={() => setDialog('link')}>
          Link to a line item
        </Button>
      )}
      <Button onClick={close}>Close</Button>
      {data.link.matchedBy === 'manual' && (
        <Button onClick={() => setDialog('unlink')}>Unlink</Button>
      )}
    </>
  ) : (
    <Button onClick={close}>Close</Button>
  );

  return (
    <>
      <SidePanel
        open
        onOpenChange={(open) => !open && close()}
        title={data?.name ?? 'GSAP transaction'}
        description={data ? `${TYPE_WORDS[data.type]} · ${data.id}` : undefined}
        footer={footer}
      >
        {data ? (
          <Details transaction={data} />
        ) : failure?.kind === 'notFound' ? (
          <p className={styles.muted}>{TRANSACTION_GONE}</p>
        ) : failure?.kind === 'forbidden' ? (
          <p className={styles.muted}>{NO_ACCESS}</p>
        ) : transaction.isError ? (
          <ErrorState error={transaction.error} onRetry={() => void transaction.refetch()} />
        ) : (
          <Skeleton lines={8} height="18px" />
        )}
      </SidePanel>
      {data && dialog === 'link' && (
        <LinkTransactionDialog transaction={data} onClose={() => setDialog(null)} />
      )}
      {data && dialog === 'unlink' && (
        <UnlinkDialog transaction={data} onClose={() => setDialog(null)} />
      )}
    </>
  );
}

function Details({ transaction }: { transaction: FinancialTransaction }) {
  const { date, dateTime, money, displayCurrency } = useFormatters();
  const t = transaction;
  const day = (timestamp: string) => date(localDate(timestamp));
  const poReference = t.poNumber ? [t.poNumber, t.poLineNumber].filter(Boolean).join(' / ') : '';

  return (
    <div className={styles.body}>
      <section aria-labelledby="transaction-fields" className={styles.section}>
        <h3 id="transaction-fields" className={styles.heading}>
          From GSAP
        </h3>
        <dl className={styles.list}>
          <Item label="Document number">{t.documentNumber}</Item>
          <Item label="Line">{t.lineNumber || '—'}</Item>
          {t.type !== 'purchaseOrder' && (
            <Item label="PO number / PO line">{poReference || '—'}</Item>
          )}
          <Item label="Posting date">{date(t.postingDate)}</Item>
          {t.accrualMonth && <Item label="Accrual month">{monthText(t.accrualMonth)}</Item>}
          <Item label="Amount">
            <span className={styles.money}>{money(t.amount)}</span>
            {t.currencyMissing && (
              <span className={styles.warning}>{currencyMissingText(t.amount.currency)}</span>
            )}
          </Item>
          {t.amount.currency !== 'USD' && (
            <Item label="Amount in USD">
              <span className={styles.money}>{money(t.amountUsd)}</span>
            </Item>
          )}
          {displayCurrency !== 'USD' && displayCurrency !== t.amount.currency && (
            <Item label={`Amount in ${displayCurrency}`}>
              <span className={styles.money}>{money(t.amountDisplay)}</span>
            </Item>
          )}
          <Item label="Vendor">{t.vendor ? `${t.vendor.code} · ${t.vendor.name}` : '—'}</Item>
          <Item label="Description">{t.description || '—'}</Item>
          <Item label="SWBS code">{t.swbsCode || '—'}</Item>
          <Item label="HX key">{t.trackingKey || '—'}</Item>
          <Item label="Line item ID as sent">{t.sentLineItemId || '—'}</Item>
        </dl>
      </section>

      <section aria-labelledby="transaction-link" className={styles.section}>
        <h3 id="transaction-link" className={styles.heading}>
          Link
        </h3>
        <dl className={styles.list}>
          <Item label="Linked to">
            <LinkedPath transaction={t} />
          </Item>
          <Item label={t.link.state === 'notLinked' ? 'Reason' : 'Matched by'}>
            {matchedByText(t, day) ?? '—'}
          </Item>
        </dl>
        {t.link.state !== 'notLinked' && t.link.matchedBy && t.link.matchedBy !== 'manual' && (
          <p className={styles.note}>
            <Info aria-hidden="true" />
            {ruleLinkedNote(matchedByText(t, day) ?? '')}
          </p>
        )}
      </section>

      <section aria-labelledby="transaction-source" className={styles.section}>
        <h3 id="transaction-source" className={styles.heading}>
          Source
        </h3>
        <dl className={styles.list}>
          <Item label="From import">
            {`${t.import.feedName}${t.import.fileName ? ` '${t.import.fileName}'` : ''} on ${dateTime(t.import.importedAt)} by ${t.import.importedBy.name}`}
          </Item>
          <Item label="Added">{dateTime(t.createdAt)}</Item>
          <Item label="Last updated">{`${dateTime(t.updatedAt)} by ${t.updatedBy.name}`}</Item>
        </dl>
      </section>
    </div>
  );
}

function Item({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className={styles.item}>
      <dt>{label}</dt>
      <dd>{children}</dd>
    </div>
  );
}

/** Plan › campaign › initiative › activity › line item, a link per level; or the state's chip. */
function LinkedPath({ transaction }: { transaction: FinancialTransaction }) {
  const { link } = transaction;
  if (link.state === 'notLinked') {
    return <StatusChip tone="review">{NOT_LINKED}</StatusChip>;
  }
  if (link.state === 'lineItemDeleted') {
    return (
      <span className={styles.stack}>
        <StatusChip tone="neutral">{LINE_ITEM_IN_BIN}</StatusChip>
        {link.reason && <span className={styles.muted}>{link.reason}</span>}
      </span>
    );
  }
  const planId = link.path.find((record) => record.type === 'plan')?.id;

  return (
    <nav aria-label="Linked to" className={styles.path}>
      {link.path.map((record, index) => (
        <Fragment key={record.id}>
          {index > 0 && <ChevronRight aria-hidden="true" className={styles.chevron} />}
          <Link to={pathHref(record, planId)}>
            {record.type === 'lineItem' ? recordLabel(record) : record.name}
          </Link>
        </Fragment>
      ))}
    </nav>
  );
}

function pathHref(record: FinancialRecordRef, planId: string | undefined): string {
  return recordHref(record.type, record.id, planId);
}

/** "Unlink 'PO 4500001004 / 10' from 'Station signage (L-1007)'?" — then Not linked again. */
function UnlinkDialog({
  transaction,
  onClose,
}: {
  transaction: FinancialTransaction;
  onClose: () => void;
}) {
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const unlink = useUnlinkTransaction();
  const lineItem = transaction.link.path.find((record) => record.type === 'lineItem');
  const lineItemText = lineItem ? recordLabel(lineItem) : (transaction.link.lineItemId ?? '');

  async function confirm() {
    if (unlink.isPending) {
      return;
    }
    try {
      await unlink.mutateAsync({ id: transaction.id, version: transaction.version });
      show({ message: `'${transaction.name}' unlinked from '${lineItemText}'` });
    } catch (error) {
      const failure = error instanceof ApiError ? error : new ApiError('other');
      if (failure.kind !== 'signInRequired' && failure.kind !== 'sessionExpired') {
        show({ message: conflictMessage(failure, 'GSAP transaction', dateFormat) });
      }
    } finally {
      onClose();
    }
  }

  return (
    <ConfirmDialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={unlinkQuestion(transaction.name, lineItemText)}
      confirmLabel="Unlink"
      busy={unlink.isPending}
      onConfirm={() => void confirm()}
    >
      It goes back to Not linked, with the reason GSAP data gives.
    </ConfirmDialog>
  );
}
