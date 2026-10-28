import type { components } from '@/api/generated/schema';

type SwbsStatus = components['schemas']['SwbsStatus'];

export const SWBS_TEXT: Record<SwbsStatus, string> = {
  notRequested: 'Not requested',
  waitingForApproval: 'Waiting for activity approval',
  readyToSend: 'Ready to send to GSAP',
  received: 'SWBS code received',
};

export const SWBS_EXPLANATION: Record<SwbsStatus, string> = {
  notRequested: "'Request SWBS code from GSAP' is off, so no code will be requested.",
  waitingForApproval: 'The request is sent once the activity is approved.',
  readyToSend: 'Sent with the next run of the simulated GSAP.',
  received: "The simulated GSAP returned this line item's SWBS code.",
};

export const SWBS_CARD_TITLE = 'SWBS request';
export const NOT_RECEIVED = 'Not received yet';
export const SIMULATED_GSAP = 'Simulated GSAP';

export const SWBS_LABEL = {
  code: 'SWBS code',
  source: 'Source',
  requested: 'Requested',
  received: 'Received',
  hxKey: 'HX key',
} as const;

export const HX_HINT = 'Generated from the line-item number';
export const HX_COPY = 'Copy';
export const HX_COPIED = 'HX key copied';
export const HX_COPY_FAILED = "Couldn't copy the HX key — select it and copy it yourself.";

export const SWBS_LATER =
  'Filled by the simulated GSAP once the activity is approved and the request is sent';
export const SWBS_HELP =
  'MAPP asks GSAP for an SWBS code once the activity is approved; until GSAP is connected, a simulated GSAP answers.';
export const SWBS_KEPT_ON =
  'This line item already has its SWBS code from the simulated GSAP, so the request stays on.';

export function copyHxLabel(hxKey: string): string {
  return `Copy HX key ${hxKey}`;
}
