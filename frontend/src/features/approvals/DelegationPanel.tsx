import { CalendarX, Pencil, Plus, UserRoundX } from 'lucide-react';
import { useId, useState } from 'react';
import type { ReactNode } from 'react';
import { Button } from '@/ui/Button/Button';
import { CheckboxField } from '@/ui/CheckboxField/CheckboxField';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { SidePanel } from '@/ui/SidePanel/SidePanel';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import { useToast } from '@/ui/Toast/useToast';
import type { StatusTone } from '@/ui/StatusChip/StatusChip';
import { useFormatters } from '@/format/useFormatters';
import { DelegationForm } from './DelegationForm';
import { useCancelDelegation, useDelegations } from './delegationsApi';
import type { ApprovalDelegation, DelegationScope } from './delegationsApi';
import { asApiError, isSessionLost } from './messages';
import styles from './DelegationPanel.module.css';

/** How many ended and cancelled delegations show before "Show all". */
const ENDED_SHOWN = 10;
const EMPTY = "You haven't delegated your approvals. Add a delegation when you will be away.";

const STATES: Record<ApprovalDelegation['state'], { word: string; tone: StatusTone }> = {
  current: { word: 'Current', tone: 'active' },
  planned: { word: 'Planned', tone: 'neutral' },
  ended: { word: 'Ended', tone: 'neutral' },
  cancelled: { word: 'Cancelled', tone: 'neutral' },
};

export interface DelegationPanelProps {
  onClose: () => void;
}

type Open = { kind: 'list' } | { kind: 'form'; delegation?: ApprovalDelegation };

/**
 * "Approval delegation" (017 US4, FR-022 – FR-024), opened by **Delegate my approvals** on the
 * approvals page: the delegations in three groups — Current, Planned, Ended and cancelled (the last
 * 10, then Show all) — each "<person> → <delegate>", its days, its state and note, with **Edit** and
 * **Cancel delegation** while it may still change. **Add delegation** opens the form in the panel's
 * place. Someone who may manage everyone's delegations sees the Everyone switch.
 */
export function DelegationPanel({ onClose }: DelegationPanelProps) {
  const [everyone, setEveryone] = useState(false);
  const [open, setOpen] = useState<Open>({ kind: 'list' });
  const [cancelling, setCancelling] = useState<ApprovalDelegation | null>(null);
  const mine = useDelegations('mine');
  const scope: DelegationScope = everyone ? 'all' : 'mine';
  const all = useDelegations('all', everyone);
  const list = everyone ? all : mine;
  const canManageOthers = mine.data?.canManageOthers ?? false;

  if (open.kind === 'form') {
    return (
      <DelegationForm
        delegation={open.delegation}
        canManageOthers={canManageOthers}
        scope={scope}
        onClose={() => setOpen({ kind: 'list' })}
      />
    );
  }

  return (
    <>
      <SidePanel
        open
        onOpenChange={(isOpen) => {
          if (!isOpen) {
            onClose();
          }
        }}
        title="Approval delegation"
        description="Who approves for whom while someone is away."
        footer={
          <>
            <Button variant="primary" icon={<Plus />} onClick={() => setOpen({ kind: 'form' })}>
              Add delegation
            </Button>
            <Button onClick={onClose}>Close</Button>
          </>
        }
      >
        <div className={styles.body}>
          {canManageOthers && (
            <CheckboxField
              label="Everyone"
              name="everyone"
              hint="Show everyone's delegations, not only yours."
              checked={everyone}
              onChange={setEveryone}
            />
          )}
          {list.data ? (
            list.data.items.length === 0 ? (
              <EmptyState icon={<UserRoundX />} title={EMPTY} />
            ) : (
              <Groups
                items={list.data.items}
                onEdit={(delegation) => setOpen({ kind: 'form', delegation })}
                onCancel={setCancelling}
              />
            )
          ) : list.isError ? (
            <ErrorState error={list.error} onRetry={() => void list.refetch()} />
          ) : (
            <Skeleton lines={6} height="18px" />
          )}
        </div>
      </SidePanel>
      {cancelling && (
        <CancelDelegationDialog delegation={cancelling} onClose={() => setCancelling(null)} />
      )}
    </>
  );
}

interface GroupsProps {
  items: ApprovalDelegation[];
  onEdit: (delegation: ApprovalDelegation) => void;
  onCancel: (delegation: ApprovalDelegation) => void;
}

/** Current, Planned, then Ended and cancelled; a group without delegations is left out. */
function Groups({ items, onEdit, onCancel }: GroupsProps) {
  const [showAll, setShowAll] = useState(false);
  const ended = items.filter((item) => item.state === 'ended' || item.state === 'cancelled');
  const groups = [
    { title: 'Current', items: items.filter((item) => item.state === 'current') },
    { title: 'Planned', items: items.filter((item) => item.state === 'planned') },
    { title: 'Ended and cancelled', items: showAll ? ended : ended.slice(0, ENDED_SHOWN) },
  ].filter((group) => group.items.length > 0);

  return (
    <>
      {groups.map((group) => (
        <Group key={group.title} title={group.title}>
          {group.items.map((delegation) => (
            <Item
              key={delegation.id}
              delegation={delegation}
              onEdit={() => onEdit(delegation)}
              onCancel={() => onCancel(delegation)}
            />
          ))}
          {group.title === 'Ended and cancelled' && !showAll && ended.length > ENDED_SHOWN && (
            <li>
              <Button variant="tertiary" size="sm" onClick={() => setShowAll(true)}>
                {`Show all (${ended.length})`}
              </Button>
            </li>
          )}
        </Group>
      ))}
    </>
  );
}

function Group({ title, children }: { title: string; children: ReactNode }) {
  const id = useId();
  return (
    <section aria-labelledby={id} className={styles.group}>
      <h3 id={id} className={styles.heading}>
        {title}
      </h3>
      <ul role="list" className={styles.items}>
        {children}
      </ul>
    </section>
  );
}

interface ItemProps {
  delegation: ApprovalDelegation;
  onEdit: () => void;
  onCancel: () => void;
}

/** "Daniel Okoro → Carlos Mendes · 03 Nov – 14 Nov 2026 · Planned", the note, Edit and Cancel. */
function Item({ delegation, onEdit, onCancel }: ItemProps) {
  const { dateRange } = useFormatters();
  const state = STATES[delegation.state];
  const who = `${delegation.person.name} → ${delegation.delegate.name}`;

  return (
    <li className={styles.item}>
      <div className={styles.itemHead}>
        <p className={styles.who}>{who}</p>
        <StatusChip tone={state.tone}>{state.word}</StatusChip>
      </div>
      <p className={styles.dates}>{dateRange(delegation.startDate, delegation.endDate)}</p>
      {delegation.note && <p className={styles.note}>{delegation.note}</p>}
      {delegation.canChange && (
        <div className={styles.actions}>
          <Button
            variant="tertiary"
            size="sm"
            icon={<Pencil />}
            aria-label={`Edit delegation ${who}`}
            onClick={onEdit}
          >
            Edit
          </Button>
          <Button
            variant="tertiary"
            size="sm"
            icon={<CalendarX />}
            aria-label={`Cancel delegation ${who}`}
            onClick={onCancel}
          >
            Cancel delegation
          </Button>
        </div>
      )}
    </li>
  );
}

/**
 * "Cancel the delegation to Carlos Mendes?" — from now on the delegate no longer approves for the
 * person; decisions already made stay recorded.
 */
function CancelDelegationDialog({
  delegation,
  onClose,
}: {
  delegation: ApprovalDelegation;
  onClose: () => void;
}) {
  const cancel = useCancelDelegation();
  const { show } = useToast();
  const delegate = delegation.delegate.name;

  async function confirm() {
    if (cancel.isPending) {
      return;
    }
    try {
      await cancel.mutateAsync({ id: delegation.id, version: delegation.version });
      show({ message: `Delegation to ${delegate} cancelled` });
    } catch (error) {
      const failure = asApiError(error);
      if (!isSessionLost(failure)) {
        show({
          message:
            failure.kind === 'notFound' ? 'This delegation no longer exists.' : failure.message,
        });
      }
    }
    onClose();
  }

  return (
    <ConfirmDialog
      open
      onOpenChange={(isOpen) => {
        if (!isOpen) {
          onClose();
        }
      }}
      title={`Cancel the delegation to ${delegate}?`}
      confirmLabel="Cancel delegation"
      cancelLabel="Keep delegation"
      busy={cancel.isPending}
      onConfirm={() => void confirm()}
    >
      <p>{`${delegate} can no longer approve for ${delegation.person.name} from now on. Decisions already made stay recorded.`}</p>
    </ConfirmDialog>
  );
}
