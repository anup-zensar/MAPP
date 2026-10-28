import { plural } from '@/format/text';

export const PANEL_TITLE = 'Simulated GSAP';
export const PANEL_SUBTITLE =
  'Stands in for GSAP until live integration: it returns one SWBS code for each line item sent to it.';
export const SEND_ALL = 'Send SWBS requests to simulated GSAP';
export const SEND_NOW = 'Send to simulated GSAP now';
export const NOTHING_READY = 'No line item is waiting for an SWBS code';
export const HISTORY_TITLE = 'Runs';
export const NO_RUNS = 'No runs yet.';
export const NO_ACCESS = 'Running the simulated GSAP needs the permission to import GSAP extracts.';
export const NOTHING_SENT = 'Nothing was ready to send.';
export const CODES_FINAL = "Codes can't be changed or removed in MAPP.";

export const SOURCE_TEXT = {
  simulatedGsap: 'Simulated GSAP',
  demoData: 'Demo data (simulated GSAP)',
} as const;

/** "1 line item is ready to send", "2 line items are ready to send". */
export function readyText(count: number): string {
  return count === 1
    ? '1 line item is ready to send'
    : `${plural(count, 'line item', 'line items')} are ready to send`;
}

export function sendAllTitle(count: number): string {
  return `Send ${plural(count, 'SWBS request', 'SWBS requests')} to simulated GSAP?`;
}

export function sendNowTitle(name: string): string {
  return `Send ${name} to simulated GSAP now?`;
}

export const SEND_ALL_TEXT = `The simulated GSAP returns one SWBS code per line item at once. ${CODES_FINAL}`;
export const SEND_NOW_TEXT = `It receives its SWBS code at once. ${CODES_FINAL}`;

export function returnedText(count: number): string {
  return `Simulated GSAP returned ${plural(count, 'SWBS code', 'SWBS codes')}`;
}

export function shownText(shown: number, total: number): string {
  return `Showing the ${shown} latest of ${total} runs`;
}

export function ranByText(when: string, who: string): string {
  return `${when} by ${who}`;
}
