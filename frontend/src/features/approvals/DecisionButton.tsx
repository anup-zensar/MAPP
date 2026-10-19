import { Check, X } from 'lucide-react';
import { useState } from 'react';
import { Button } from '@/components';
import type { ActivityApproval } from './approvalsApi';
import { DecisionDialog } from './DecisionDialog';

export interface DecisionButtonProps {
  mode: 'approve' | 'decline';
  approval: ActivityApproval;
  /** The owner's name when known (see DecisionDialog). */
  ownerName?: string | null;
  variant?: 'primary' | 'secondary';
  onDecided?: (approval: ActivityApproval) => void;
}

/** **Approve** or **Decline**, with the dialog that confirms the decision (US2). */
export function DecisionButton({
  mode,
  approval,
  ownerName,
  variant = mode === 'approve' ? 'primary' : 'secondary',
  onDecided,
}: DecisionButtonProps) {
  const [open, setOpen] = useState(false);

  return (
    <>
      <Button
        variant={variant}
        icon={mode === 'approve' ? <Check /> : <X />}
        onClick={() => setOpen(true)}
      >
        {mode === 'approve' ? 'Approve' : 'Decline'}
      </Button>
      {open && (
        <DecisionDialog
          mode={mode}
          approval={approval}
          ownerName={ownerName}
          onClose={() => setOpen(false)}
          onDecided={onDecided}
        />
      )}
    </>
  );
}
