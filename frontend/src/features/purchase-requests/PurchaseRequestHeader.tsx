import { Download, Pencil } from 'lucide-react';
import { useState } from 'react';
import { useNavigate } from 'react-router';
import { asApiError } from '@/api/errors';
import { showHistory } from '@/features/activity-log';
import { allows, gateItem } from '@/session/gate-action';
import { Button, LinkButton } from '@/ui/Button/Button';
import { CompletenessChip } from '@/ui/CompletenessChip/CompletenessChip';
import { LevelTag } from '@/ui/LevelTag/LevelTag';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { PersonLabel } from '@/ui/PersonLabel/PersonLabel';
import { purchaseRequestHref } from '@/ui/record-links';
import { useToast } from '@/ui/Toast/useToast';
import { downloadPurchaseRequestForm } from './api';
import type { PurchaseRequest } from './api';
import { ClosePurchaseRequestDialog } from './ClosePurchaseRequestDialog';
import { DeletePurchaseRequestDialog } from './DeletePurchaseRequestDialog';
import { GsapOrderButton } from './GsapOrderButton';
import { fieldTarget, missingOf, purchaseRequestBreadcrumb } from './page-links';
import { CLOSE_PR, DELETE_PR, DOWNLOAD_FORM, EDIT, RESEND_FORM, VIEW_HISTORY } from './page-text';
import { PurchaseRequestStatusChip } from './PurchaseRequestStatusChip';
import { ResendFormDialog } from './ResendFormDialog';
import { SubmitPurchaseRequestButton } from './SubmitPurchaseRequestDialog';
import styles from './PurchaseRequestPage.module.css';

type OpenDialog = 'close' | 'delete' | 'resend' | null;

export function PurchaseRequestHeader({
  purchaseRequest: pr,
}: {
  purchaseRequest: PurchaseRequest;
}) {
  const navigate = useNavigate();
  const { show } = useToast();
  const [open, setOpen] = useState<OpenDialog>(null);
  const href = purchaseRequestHref(pr.id);
  const [latest] = pr.routings;

  function selectField(field: string) {
    void navigate(fieldTarget(field, href));
  }

  async function downloadLatest() {
    if (latest) {
      await downloadPurchaseRequestForm(pr.id, latest.id, 'xlsx').catch((error: unknown) =>
        show({ message: asApiError(error).message }),
      );
    }
  }

  const dialog = (name: OpenDialog) => ({
    purchaseRequest: pr,
    open: open === name,
    onOpenChange: (next: boolean) => setOpen(next ? name : null),
  });

  return (
    <>
      <PageHeader
        breadcrumb={purchaseRequestBreadcrumb(pr)}
        title={pr.name}
        meta={
          <>
            <LevelTag level="purchaseRequest" />
            <PurchaseRequestStatusChip status={pr.status} changePending={pr.changePending} />
            <CompletenessChip missing={missingOf(pr)} onSelectField={selectField} />
            <span className={styles.fact}>{pr.id}</span>
            {pr.owner && <PersonLabel person={pr.owner} />}
          </>
        }
        moreActions={[
          ...gateItem({ label: RESEND_FORM, onSelect: () => setOpen('resend') }, pr.access.resend),
          ...gateItem({ label: CLOSE_PR, onSelect: () => setOpen('close') }, pr.access.close),
          { label: VIEW_HISTORY, onSelect: () => showHistory() },
          ...gateItem(
            {
              label: DELETE_PR,
              tone: 'danger',
              separatorBefore: true,
              onSelect: () => setOpen('delete'),
            },
            pr.access.delete,
          ),
        ]}
        actions={
          <>
            <SubmitPurchaseRequestButton purchaseRequest={pr} />
            <GsapOrderButton purchaseRequest={pr} />
            {latest && (
              <Button icon={<Download />} onClick={() => void downloadLatest()}>
                {DOWNLOAD_FORM}
              </Button>
            )}
            {allows(pr.access.edit) && (
              <LinkButton to={`${href}/edit`} variant="primary" icon={<Pencil />}>
                {EDIT}
              </LinkButton>
            )}
          </>
        }
      />
      <ResendFormDialog {...dialog('resend')} />
      <ClosePurchaseRequestDialog {...dialog('close')} />
      <DeletePurchaseRequestDialog {...dialog('delete')} />
    </>
  );
}
