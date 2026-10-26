import { BellRing } from 'lucide-react';
import { useState } from 'react';
import { Button } from '@/ui/Button/Button';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { useToast } from '@/ui/Toast/useToast';
import { asApiError, isSessionLost } from '@/api/errors';
import { useRunEscalations } from './api';
import { escalationRunText, notEscalatedText } from './decision-messages';

/** Long enough to read a list of activities that could not be escalated. */
const NOT_ESCALATED_MS = 10_000;

/**
 * **Run reminders and escalations** (017 US5, FR-028), on the Monitoring tab for someone who may
 * change approval workflows: asks first, then sends what is due now — reminders once a level has
 * waited the reminder days, the backup approver once it has waited the escalation days — and says
 * what was sent; a second message names the levels no backup approver could take.
 */
export function RunEscalationsButton() {
  const run = useRunEscalations();
  const { show } = useToast();
  const [asking, setAsking] = useState(false);

  async function confirm() {
    if (run.isPending) {
      return;
    }
    try {
      const result = await run.mutateAsync();
      setAsking(false);
      show({ message: escalationRunText(result) });
      if (result.notEscalated.length > 0) {
        show({ message: notEscalatedText(result.notEscalated), durationMs: NOT_ESCALATED_MS });
      }
    } catch (error) {
      setAsking(false);
      const failure = asApiError(error);
      if (!isSessionLost(failure)) {
        show({ message: failure.message });
      }
    }
  }

  return (
    <>
      <Button
        icon={<BellRing />}
        aria-busy={run.isPending || undefined}
        onClick={() => setAsking(true)}
      >
        {run.isPending ? 'Running…' : 'Run reminders and escalations'}
      </Button>
      <ConfirmDialog
        open={asking}
        onOpenChange={(open) => {
          if (!open && !run.isPending) {
            setAsking(false);
          }
        }}
        title="Run reminders and escalations now?"
        confirmLabel="Run now"
        cancelLabel="Cancel"
        busy={run.isPending}
        onConfirm={() => void confirm()}
      >
        <p>
          Approvers get a reminder once a level has waited the reminder days, and the backup
          approver takes over once it has waited the escalation days. Each happens once per level.
        </p>
      </ConfirmDialog>
    </>
  );
}
