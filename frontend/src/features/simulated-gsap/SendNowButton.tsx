import { Send } from 'lucide-react';
import { useState } from 'react';
import { Button } from '@/ui/Button/Button';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { useSendSwbsRequests } from './api';
import { SEND_NOW, SEND_NOW_TEXT, sendNowTitle } from './simulated-gsap-text';
import styles from './SimulatedGsapPanel.module.css';

export interface SendNowButtonProps {
  lineItemId: string;
  lineItemName: string;
}

/** Sends one line item to the simulated GSAP after confirmation (030 FR-015); its page then shows the code. */
export function SendNowButton({ lineItemId, lineItemName }: SendNowButtonProps) {
  const send = useSendSwbsRequests();
  const [confirming, setConfirming] = useState(false);

  function handleConfirm() {
    send.mutate([lineItemId], { onSettled: () => setConfirming(false) });
  }

  return (
    <>
      <Button icon={<Send />} onClick={() => setConfirming(true)}>
        {SEND_NOW}
      </Button>
      {send.error && (
        <p role="alert" className={styles.problem}>
          {send.error.message}
        </p>
      )}
      <ConfirmDialog
        open={confirming}
        onOpenChange={setConfirming}
        title={sendNowTitle(lineItemName)}
        confirmLabel={SEND_NOW}
        onConfirm={handleConfirm}
        busy={send.isPending}
      >
        <p>{SEND_NOW_TEXT}</p>
      </ConfirmDialog>
    </>
  );
}
