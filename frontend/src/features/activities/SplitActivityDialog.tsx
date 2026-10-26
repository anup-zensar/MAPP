import { CircleAlert } from 'lucide-react';
import { useEffect, useId, useState } from 'react';
import { useNavigate } from 'react-router';
import { ApiError, conflictMessage } from '@/api/errors';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { FormDialog } from '@/ui/FormDialog/FormDialog';
import { FailureMessage } from '@/ui/FormPanel/FormPanel';
import { MoneyField } from '@/ui/MoneyField/MoneyField';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { TextField } from '@/ui/TextField/TextField';
import { useToast } from '@/ui/Toast/useToast';
import { centsInput, moneyText } from '@/features/line-items';
import { parseMoneyInput, toCents } from '@/format/decimal';
import { useFormatters } from '@/format/useFormatters';
import type { Activity } from './activitiesApi';
import { ActivityStatusChip } from '@/ui/ActivityStatusChip/ActivityStatusChip';
import { activityHref } from './activityText';
import { DestinationList } from './DestinationList';
import { useSplitActivity, useSplitOptions, useSplitPreview } from './splitApi';
import type { ActivitySplitRequest, ActivitySplitSide } from './splitApi';
import styles from './SplitActivityDialog.module.css';
import { goalsStayText, NOTHING_CHOSEN, REAPPROVAL, splitToast, splitTitle } from './splitText';

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
  const listId = useId();
  const code = options?.planCurrency ?? '';
  // The new activity may stay in the activity's own initiative (the default), which Move refuses.
  const destinations = (options?.destinations ?? []).map((item) =>
    item.current ? { ...item, moveProblem: null } : item,
  );
  const current = destinations.find((item) => item.current);
  // Typed or chosen values; until then the suggested name and the activity's own initiative.
  const [typedName, setName] = useState<string | null>(null);
  const [chosenInitiative, setInitiativeId] = useState<string | null>(null);
  const name = typedName ?? options?.suggestedName ?? '';
  const initiativeId = chosenInitiative ?? current?.initiative.id ?? '';
  const [ticked, setTicked] = useState<string[]>([]);
  // The typed amount; null while the amount follows the ticked line items.
  const [typedAmount, setTypedAmount] = useState<string | null>(null);
  const [problems, setProblems] = useState<string[]>([]);
  const [failure, setFailure] = useState<string | null>(null);

  const lineItems = options?.lineItems ?? [];
  const tickedCents = lineItems
    .filter((item) => ticked.includes(item.id))
    .reduce((sum, item) => sum + toCents(item.plannedCostPlan.amount), 0n);
  const amount = typedAmount ?? (ticked.length > 0 ? centsInput(tickedCents) : '0.00');
  const parsed = amount.trim() === '' ? null : parseMoneyInput(amount);
  const nothingChosen = ticked.length === 0 && (!parsed || toCents(parsed.amount) <= 0n);

  const request: ActivitySplitRequest = {
    name: name.trim(),
    initiativeId,
    lineItemIds: ticked,
    plannedCost: parsed ? parsed.amount : null,
    restartApproval: options?.needsRestart ?? false,
    version: options?.version ?? 0,
  };
  const debounced = useDebounced(!options || nothingChosen ? null : request, PREVIEW_DELAY_MS);
  const preview = useSplitPreview(activity.id, debounced);
  const previewProblems = preview.data?.problems ?? [];

  function toggle(id: string, on: boolean) {
    setTicked((list) => (on ? [...list, id] : list.filter((each) => each !== id)));
    setProblems([]);
  }

  async function submit() {
    if (!options) {
      return;
    }
    if (nothingChosen) {
      setProblems([NOTHING_CHOSEN]);
      return;
    }
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
  }

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
      onSubmit={() => void submit()}
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
          <TextField
            label="New activity name"
            name="split-name"
            marking="Required"
            maxLength={120}
            value={name}
            onChange={setName}
          />
          <DestinationList
            legend="Initiative"
            destinations={destinations}
            purpose="move"
            value={initiativeId}
            onChange={setInitiativeId}
          />
          <fieldset className={styles.items} aria-describedby={`${listId}-hint`}>
            <legend className={styles.legend}>Line items to move</legend>
            <p id={`${listId}-hint`} className={styles.hint}>
              They keep their IDs, values and spreads. Amounts in {code}, the plan currency.
            </p>
            {lineItems.length === 0 ? (
              <p className={styles.hint}>This activity has no line items.</p>
            ) : (
              <ul className={styles.list}>
                {lineItems.map((item) => {
                  const id = `${listId}-${item.id}`;
                  return (
                    <li key={item.id} className={styles.item}>
                      <input
                        id={id}
                        type="checkbox"
                        checked={ticked.includes(item.id)}
                        onChange={(event) => toggle(item.id, event.target.checked)}
                      />
                      <label htmlFor={id}>{item.name}</label>
                      <span className={styles.amount}>
                        {moneyText(item.plannedCostPlan)}
                        {item.plannedCost.currency !== item.plannedCostPlan.currency && (
                          <span className={styles.local}> ({moneyText(item.plannedCost)})</span>
                        )}
                      </span>
                    </li>
                  );
                })}
              </ul>
            )}
          </fieldset>
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
              setTypedAmount(text);
              setProblems([]);
            }}
            note={
              typedAmount === null && ticked.length > 0 ? 'The ticked line items’ total' : undefined
            }
          />
          <p className={styles.hint}>{goalsStayText(activity.name)}</p>
          {options.needsRestart && (
            <p className={styles.warning}>
              <CircleAlert aria-hidden="true" />
              {REAPPROVAL}
            </p>
          )}
          <Preview
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

/** Both activities as they will be (FR-027), side by side from 600 px. */
function Preview({
  original,
  created,
  waiting,
}: {
  original: ActivitySplitSide | null;
  created: ActivitySplitSide | null;
  waiting: 'nothing' | 'loading' | null;
}) {
  return (
    <section aria-label="After the split" className={styles.preview}>
      <h3 className={styles.legend}>After the split</h3>
      {waiting === 'nothing' ? (
        <p className={styles.hint}>Choose line items or an amount to see both activities.</p>
      ) : waiting === 'loading' || !original || !created ? (
        <Skeleton lines={3} />
      ) : (
        <div className={styles.sides} aria-live="polite">
          <Side side={original} />
          <Side side={created} isNew />
        </div>
      )}
    </section>
  );
}

function Side({ side, isNew = false }: { side: ActivitySplitSide; isNew?: boolean }) {
  const negative = side.unassignedState === 'exceeded';
  return (
    <div className={styles.side}>
      <p className={styles.sideName}>
        {isNew ? `New: ${side.name}` : side.name}
        <span className={styles.sideInitiative}>{side.initiativeName}</span>
      </p>
      <dl className={styles.facts}>
        <div>
          <dt>Planned cost</dt>
          <dd>{side.plannedCost ? moneyText(side.plannedCost) : '—'}</dd>
        </div>
        <div>
          <dt>Unassigned planned cost</dt>
          <dd className={negative ? styles.negative : undefined}>
            {side.unassignedPlannedCost ? moneyText(side.unassignedPlannedCost) : '—'}
            {negative && ' — line items exceed the planned cost'}
          </dd>
        </div>
        <div>
          <dt>Line items</dt>
          <dd>{side.lineItemNames.length > 0 ? side.lineItemNames.join(', ') : 'None'}</dd>
        </div>
        <div>
          <dt>Status</dt>
          <dd>
            <ActivityStatusChip status={side.status} />
          </dd>
        </div>
      </dl>
    </div>
  );
}

/** The value once it has stayed the same for `delay` ms. */
function useDebounced<T>(value: T, delay: number): T {
  const [settled, setSettled] = useState(value);
  const key = JSON.stringify(value);
  useEffect(() => {
    const timer = window.setTimeout(() => setSettled(JSON.parse(key) as T), delay);
    return () => window.clearTimeout(timer);
  }, [key, delay]);
  return settled;
}
