import { Undo2 } from 'lucide-react';
import { useState } from 'react';
import { Button } from '@/ui/Button/Button';
import type { ActivityApproval } from './approvalsApi';
import { RecallDialog } from './RecallDialog';

export interface RecallButtonProps {
  /** A Submitted activity's approval (its open round is the one recalled). */
  approval: ActivityApproval;
}

/** **Recall submission**, with the dialog that confirms it (US5, FR-021). */
export function RecallButton({ approval }: RecallButtonProps) {
  const [open, setOpen] = useState(false);

  return (
    <>
      <Button icon={<Undo2 />} onClick={() => setOpen(true)}>
        Recall submission
      </Button>
      {open && <RecallDialog approval={approval} onClose={() => setOpen(false)} />}
    </>
  );
}
