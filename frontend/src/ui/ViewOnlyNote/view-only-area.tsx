import type { ReactNode } from 'react';
import { HidesUnavailable } from './view-only-context';

export interface ViewOnlyAreaProps {
  active: boolean;
  children: ReactNode;
}

// Where the person may change nothing, menus leave out what they may not do; the View only note says why.
export function ViewOnlyArea({ active, children }: ViewOnlyAreaProps) {
  return <HidesUnavailable value={active}>{children}</HidesUnavailable>;
}
