import { useState } from 'react';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import type { ActivityApproval } from './api';
import { DecisionButton } from './DecisionButton';
import { DecisionDialog } from './DecisionDialog';
import { approveRefusal } from './decision-role';
import { RefusedApprove } from './OverPlannedCost';

export interface DecisionButtonsProps {
  approval: ActivityApproval;
  /** Approve is the page's or panel's one main action (an approver's own view). */
  primary?: boolean;
  onDecided?: (approval: ActivityApproval) => void;
  /** The element showing why Approve is refused, when the caller shows it (038 FR-020). */
  refusalId?: string;
}

/**
 * **Decline** then **Approve**, side by side, so Approve ends the row (design system §2.5); Approve
 * is shown disabled with its reason when it is refused (038 FR-020).
 */
export function DecisionButtons({
  approval,
  primary = false,
  onDecided,
  refusalId,
}: DecisionButtonsProps) {
  const { approve, decline } = approval.actions;
  const refusal = approveRefusal(approval.actions);
  return (
    <>
      {decline.available && (
        <DecisionButton mode="decline" approval={approval} onDecided={onDecided} />
      )}
      {refusal && <RefusedApprove reason={refusal} reasonId={refusalId} />}
      {approve.available && (
        <DecisionButton
          mode="approve"
          approval={approval}
          variant={primary ? 'primary' : 'secondary'}
          onDecided={onDecided}
        />
      )}
    </>
  );
}

export interface OwnerDecisionMenuProps {
  approval: ActivityApproval;
  onDecided?: (approval: ActivityApproval) => void;
}

/**
 * The owner's way to decide while the rules still allow it: a ⋯ menu ("Actions for <activity>")
 * with Approve and Decline, each saying that they own it; the dialog repeats the warning.
 */
export function OwnerDecisionMenu({ approval, onDecided }: OwnerDecisionMenuProps) {
  const [mode, setMode] = useState<'approve' | 'decline' | null>(null);
  const { approve, decline } = approval.actions;
  const hint = approval.viewer.isOwner ? 'You own this activity.' : 'You submitted this activity.';

  return (
    <>
      <ActionsMenu
        label={`approval of ${approval.activityName}`}
        items={[
          ...(approve.available
            ? [{ label: 'Approve', hint, onSelect: () => setMode('approve') }]
            : []),
          ...(decline.available
            ? [{ label: 'Decline', hint, onSelect: () => setMode('decline') }]
            : []),
        ]}
      />
      {mode && (
        <DecisionDialog
          mode={mode}
          approval={approval}
          onClose={() => setMode(null)}
          onDecided={onDecided}
        />
      )}
    </>
  );
}
