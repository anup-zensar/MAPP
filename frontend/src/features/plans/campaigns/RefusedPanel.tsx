import type { Availability } from '@/session/access';
import { NO_ACCESS_DEFAULT } from '@/ui/NoAccess/NoAccess';
import { GoneMessage, StatePanel } from './PanelFrame';

export interface RefusedPanelProps {
  title: string;
  closeTo: string;
  availability: Availability | undefined;
}

/** A form panel the person may not use here: the server's reason and **Back to the plan**. */
export function RefusedPanel({ title, closeTo, availability }: RefusedPanelProps) {
  return (
    <StatePanel title={title} closeTo={closeTo}>
      <GoneMessage message={availability?.reason || NO_ACCESS_DEFAULT} />
    </StatePanel>
  );
}
