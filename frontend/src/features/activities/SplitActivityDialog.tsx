import { zodResolver } from '@hookform/resolvers/zod';
import { CircleAlert } from 'lucide-react';
import { useState } from 'react';
import { useForm, useWatch } from 'react-hook-form';
import { useNavigate } from 'react-router';
import { ApiError, conflictMessage } from '@/api/errors';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { FormDialog } from '@/ui/FormDialog/FormDialog';
import { FormField } from '@/ui/FormField/FormField';
import { FailureMessage } from '@/ui/FormPanel/FormPanel';
import { MoneyField } from '@/ui/MoneyField/MoneyField';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { TextField } from '@/ui/TextField/TextField';
import { useToast } from '@/ui/Toast/useToast';
import { CHECK_ON_SAVE } from '@/ui/useFormProblems';
import { centsInput, moneyText } from '@/features/line-items';
import { parseMoneyInput, toCents } from '@/format/decimal';
import { useFormatters } from '@/format/useFormatters';
import type { Activity } from './api';
import { activityHref } from './activity-text';
import { DestinationList } from './DestinationList';
import { isNothingChosen, splitSchema } from './schemas';
import type { SplitValues } from './schemas';
import { useSplitActivity, useSplitOptions, useSplitPreview } from './split-api';
import type { ActivitySplitRequest } from './split-api';
import styles from './SplitActivityDialog.module.css';
import { goalsStayText, REAPPROVAL, splitToast, splitTitle } from './split-text';
import { SplitLineItems } from './SplitLineItems';
import { SplitPreview } from './SplitPreview';
import { useDebounced } from './useDebounced';

/** How long the dialog waits after the last change before it asks for the preview. */
const PREVIEW_DELAY_MS = 300;

export interface SplitActivityDialogProps {
  activity: Activity;
  onClose: () => void;
}

/**
 * **More › Split activity** (016 US4, FR-026 – FR-029): the new activity's name, its initiative
 * (Move's availability and reasons), the line items to move with their plan-currency amounts, and
 * the planned cost to move — the sum of the ticked line items until the person types another
 * amount. Before confirming, the API's preview shows both activities as they will be; every
 * problem shows at once. A submitted or approved activity says it will need approval again.
 */
export function SplitActivityDialog({ activity, onClose }: SplitActivityDialogProps) {
  const query = useSplitOptions(activity.id);
  const options = query.data;
  const navigate = useNavigate();
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const split = useSplitActivity();
  const code = options?.planCurrency ?? '';
  // The new activity may stay in the activity's own initiative (the default), which Move refuses.
  const destinations = (options?.destinations ?? []).map((item) =>
    item.current ? { ...item, moveProblem: null } : item,
  );
  const current = destinations.find((item) => item.current);
  // Until typed or chosen (null): the suggested name, the own initiative, the ticked items' total.
  const form = useForm<SplitValues>({
    resolver: zodResolver(splitSchema),
    defaultValues: { name: null, initiativeId: null, lineItemIds: [], plannedCost: null },
    ...CHECK_ON_SAVE,
  });
  const values = useWatch({ control: form.control, compute: (all) => all });
  const [problems, setProblems] = useState<string[]>([]);
  const [failure, setFailure] = useState<string | null>(null);

  const lineItems = options?.lineItems ?? [];
  const ticked = values.lineItemIds;
  const tickedCents = lineItems
    .filter((item) => ticked.includes(item.id))
    .reduce((sum, item) => sum + toCents(item.plannedCostPlan.amount), 0n);
  const amount = values.plannedCost ?? (ticked.length > 0 ? centsInput(tickedCents) : '0.00');
  const parsed = amount.trim() === '' ? null : parseMoneyInput(amount);
  const nothingChosen = isNothingChosen(ticked, amount);

  const request: ActivitySplitRequest = {
    name: (values.name ?? options?.suggestedName ?? '').trim(),
    initiativeId: values.initiativeId ?? current?.initiative.id ?? '',
    lineItemIds: ticked,
    plannedCost: parsed ? parsed.amount : null,
    restartApproval: options?.needsRestart ?? false,
    version: options?.version ?? 0,
  };
  const debounced = useDebounced(!options || nothingChosen ? null : request, PREVIEW_DELAY_MS);
  const preview = useSplitPreview(activity.id, debounced);
  const previewProblems = preview.data?.problems ?? [];

  const submit = form.handleSubmit(
    async () => {
      setFailure(null);
      try {
        const result = await split.mutateAsync({ id: activity.id, body: request });
        onClose();
        const created = result.newActivity;
        show({
          message: splitToast(result.activity.name, created.name),
          action: {
            label: 'View',
            altText: `Open '${created.name}' from the plan`,
            onClick: () => void navigate(activityHref(created.plan.id, created.id)),
          },
        });
      } catch (error) {
        const apiError = error instanceof ApiError ? error : new ApiError('other');
        if (apiError.kind === 'conflict') {
          setFailure(conflictMessage(apiError, 'activity', dateFormat));
        } else if (apiError.fields.length > 0) {
          setProblems(apiError.fields.map((field) => field.message));
        } else {
          setFailure(apiError.message);
        }
      }
    },
    () => {
      const message = form.getFieldState('lineItemIds').error?.message;
      setProblems(message ? [message] : []);
    },
  );

  const shownProblems = [...problems, ...previewProblems.map((problem) => problem.message)].filter(
    (message, index, all) => all.indexOf(message) === index,
  );

  return (
    <FormDialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={splitTitle(activity.name)}
      description="Move line items and part of the planned cost into a new activity of this plan."
      submitLabel="Split activity"
      onSubmit={() => options && void submit()}
      busy={split.isPending}
    >
      {!options ? (
        query.isError ? (
          <ErrorState error={query.error} onRetry={() => void query.refetch()} />
        ) : (
          <Skeleton lines={4} />
        )
      ) : (
        <div className={styles.stack}>
          {failure && <FailureMessage failure={{ message: failure }} />}
          {shownProblems.length > 0 && (
            <div role="alert" className={styles.problems}>
              {shownProblems.map((message) => (
                <p key={message}>
                  <CircleAlert aria-hidden="true" />
                  {message}
                </p>
              ))}
            </div>
          )}
          <FormField
            control={form.control}
            name="name"
            render={(field) => (
              <TextField
                label="New activity name"
                name="split-name"
                marking="Required"
                maxLength={120}
                value={field.value ?? options.suggestedName}
                onChange={field.onChange}
              />
            )}
          />
          <FormField
            control={form.control}
            name="initiativeId"
            render={(field) => (
              <DestinationList
                legend="Initiative"
                destinations={destinations}
                purpose="move"
                value={field.value ?? current?.initiative.id ?? ''}
                onChange={field.onChange}
              />
            )}
          />
          <FormField
            control={form.control}
            name="lineItemIds"
            render={(field) => (
              <SplitLineItems
                lineItems={lineItems}
                code={code}
                value={field.value}
                onChange={(ids) => {
                  field.onChange(ids);
                  setProblems([]);
                }}
              />
            )}
          />
          <FormField
            control={form.control}
            name="plannedCost"
            render={(field) => (
              <MoneyField
                label="Planned cost to move"
                name="split-amount"
                hint={
                  options.plannedCost
                    ? `From 0.00 up to ${moneyText(options.plannedCost)}, the activity's planned cost`
                    : 'The activity has no planned cost yet'
                }
                currency={code}
                value={amount}
                onChange={(text) => {
                  field.onChange(text);
                  setProblems([]);
                }}
                note={
                  field.value === null && ticked.length > 0
                    ? 'The ticked line items’ total'
                    : undefined
                }
              />
            )}
          />
          <p className={styles.hint}>{goalsStayText(activity.name)}</p>
          {options.needsRestart && (
            <p className={styles.warning}>
              <CircleAlert aria-hidden="true" />
              {REAPPROVAL}
            </p>
          )}
          <SplitPreview
            original={preview.data?.original ?? null}
            created={preview.data?.created ?? null}
            waiting={
              nothingChosen ? 'nothing' : preview.isFetching && !preview.data ? 'loading' : null
            }
          />
        </div>
      )}
    </FormDialog>
  );
}
