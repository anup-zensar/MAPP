import { zodResolver } from '@hookform/resolvers/zod';
import { Send } from 'lucide-react';
import { useId, useState } from 'react';
import { useForm } from 'react-hook-form';
import { Link } from 'react-router';
import { z } from 'zod';
import type { components } from '@/api/generated/schema';
import { asApiError, isSessionLost } from '@/api/errors';
import { ApprovalChain } from '@/features/approvals';
import { Button } from '@/ui/Button/Button';
import { FormDialog } from '@/ui/FormDialog/FormDialog';
import { FormField } from '@/ui/FormField/FormField';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { TextAreaField } from '@/ui/TextAreaField/TextAreaField';
import { useToast } from '@/ui/Toast/useToast';
import { CHECK_ON_SAVE } from '@/ui/useFormProblems';
import { useHidesUnavailable } from '@/ui/ViewOnlyNote/view-only-context';
import { missingLinks, problemLinks } from './approval-links';
import type { ProblemLink } from './approval-links';
import {
  CLOSE,
  COMMENT_LABEL,
  NEEDS_FIRST,
  RESUBMIT,
  SUBMIT,
  notReadyTitle,
  submitTitle,
  submittedToast,
  workflowText,
} from './approval-text';
import { usePurchaseRequestSubmission, useSubmitPurchaseRequest } from './api';
import type { PurchaseRequest, PurchaseRequestSubmission } from './api';
import styles from './PurchaseRequestApprovalSection.module.css';

type Availability = components['schemas']['ActionAvailability'];

const COMMENT_MAX = 2000;
const commentSchema = z.object({
  comment: z.string().max(COMMENT_MAX, 'Keep the comment to 2,000 characters.'),
});

export interface SubmitPurchaseRequestButtonProps {
  purchaseRequest: PurchaseRequest;
  /** The Approval section's own actions: shown only when available (the section says why not). */
  actions?: { submit: Availability; resubmit: Availability };
}

/** **Submit for approval** or **Re-submit** (032 FR-021, FR-023), explained when unavailable. */
export function SubmitPurchaseRequestButton({
  purchaseRequest,
  actions,
}: SubmitPurchaseRequestButtonProps) {
  const [open, setOpen] = useState(false);
  const reasonId = useId();
  const hidesUnavailable = useHidesUnavailable();
  const { submit, resubmit } = actions ?? purchaseRequest.access;
  const label = submit.available || (!resubmit.available && submit.reason) ? SUBMIT : RESUBMIT;

  if (!submit.available && !resubmit.available) {
    const reason = actions || hidesUnavailable ? '' : submit.reason || resubmit.reason;
    return reason ? (
      <span className={styles.withReason}>
        <Button icon={<Send />} disabled aria-describedby={reasonId}>
          {label}
        </Button>
        <span id={reasonId} className={styles.reason}>
          {reason}
        </span>
      </span>
    ) : null;
  }
  return (
    <>
      <Button icon={<Send />} onClick={() => setOpen(true)}>
        {label}
      </Button>
      {open && (
        <SubmitPurchaseRequestDialog
          purchaseRequest={purchaseRequest}
          onClose={() => setOpen(false)}
        />
      )}
    </>
  );
}

interface DialogProps {
  purchaseRequest: PurchaseRequest;
  onClose: () => void;
}

/**
 * What Submit would do now (032 FR-021): every gate problem at once, each linked to where it is
 * fixed, or the workflow's levels and an optional comment; a refusal that names missing fields
 * lists them all in the same place.
 */
function SubmitPurchaseRequestDialog({ purchaseRequest: pr, onClose }: DialogProps) {
  const { show } = useToast();
  const check = usePurchaseRequestSubmission(pr.id, true);
  const submit = useSubmitPurchaseRequest(pr.id);
  const [missing, setMissing] = useState<ProblemLink[] | null>(null);
  const [problem, setProblem] = useState('');
  const form = useForm({
    resolver: zodResolver(commentSchema),
    defaultValues: { comment: '' },
    ...CHECK_ON_SAVE,
  });
  const data = check.data;
  const resubmit = data ? data.kind === 'resubmit' : !pr.access.submit.available;
  const ready = Boolean(data?.available && data.ready && !missing);
  const blocked = data !== undefined && !ready;

  async function send(found: PurchaseRequestSubmission, comment: string) {
    try {
      await submit.mutateAsync({ version: found.version, comment: comment.trim() });
      show({ message: submittedToast(found.kind === 'resubmit', pr.name) });
      onClose();
    } catch (error) {
      const failure = asApiError(error);
      const commentProblem = failure.fields.find((field) => field.field === 'comment');
      if (failure.kind === 'validation' && commentProblem) {
        form.setError('comment', { message: commentProblem.message });
      } else if (failure.kind === 'notReady') {
        setMissing(missingLinks(pr.id, failure.missing));
      } else if (!isSessionLost(failure)) {
        setProblem(failure.message);
      }
    }
  }

  function handleSubmit() {
    if (check.isError) {
      void check.refetch();
    } else if (blocked) {
      onClose();
    } else if (data && !submit.isPending) {
      void form.handleSubmit(({ comment }) => send(data, comment))();
    }
  }

  return (
    <FormDialog
      open
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
      title={blocked ? notReadyTitle(pr.name) : submitTitle(resubmit, pr.name)}
      submitLabel={check.isError ? 'Try again' : blocked ? CLOSE : resubmit ? RESUBMIT : SUBMIT}
      hideCancel={blocked}
      busy={submit.isPending || (!data && !check.isError)}
      onSubmit={handleSubmit}
    >
      <div className={styles.dialogBody}>
        {check.isError ? (
          <p role="alert" className={styles.problem}>
            {asApiError(check.error).message}
          </p>
        ) : !data ? (
          <Skeleton lines={4} />
        ) : missing ? (
          <ProblemList links={missing} />
        ) : !data.available ? (
          <p>{data.reason}</p>
        ) : !data.ready ? (
          <ProblemList links={problemLinks(pr.id, data.problems)} />
        ) : (
          <>
            <p>{workflowText(data.workflow)}</p>
            {data.levels.length > 0 && <ApprovalChain upcomingLevels={data.levels} />}
            {problem && (
              <p role="alert" className={styles.problem}>
                {problem}
              </p>
            )}
            <FormField
              control={form.control}
              name="comment"
              render={(field) => (
                <TextAreaField label={COMMENT_LABEL} rows={3} maxLength={COMMENT_MAX} {...field} />
              )}
            />
          </>
        )}
      </div>
    </FormDialog>
  );
}

function ProblemList({ links }: { links: ProblemLink[] }) {
  return (
    <>
      <p>{NEEDS_FIRST}</p>
      <ul className={styles.links}>
        {links.map((link) => (
          <li key={link.key}>
            <Link to={link.href}>{link.text}</Link>
          </li>
        ))}
      </ul>
    </>
  );
}
