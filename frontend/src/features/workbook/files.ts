import { saveBlob } from '@/api/download';
import type { ImportProblem } from './api';

export const MAX_FILE_BYTES = 5 * 1024 * 1024;
const BYTE_ORDER_MARK = String.fromCharCode(0xfeff);

/** The file's content in base64, as the upload sends it. */
export function base64Of(file: Blob): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      const url = String(reader.result);
      resolve(url.slice(url.indexOf(',') + 1));
    };
    reader.onerror = () => reject(reader.error ?? new Error('The file could not be read.'));
    reader.readAsDataURL(file);
  });
}

function csvField(text: string): string {
  const safe = /^[=+\-@\t\r]/.test(text) ? `'${text}` : text;
  return /[",\n]/.test(safe) ? `"${safe.replaceAll('"', '""')}"` : safe;
}

export function problemsCsv(problems: ImportProblem[]): string {
  const rows = [['Sheet', 'Row', 'Column', 'Value found', 'Reason']];
  for (const p of problems) {
    rows.push([p.sheet, String(p.row), p.column, p.value, p.reason]);
  }
  return `${BYTE_ORDER_MARK}${rows.map((row) => row.map(csvField).join(',')).join('\n')}\n`;
}

export function downloadProblems(problems: ImportProblem[], fileName: string) {
  const name = `Problems – ${fileName.replace(/\.[^.]+$/, '')}.csv`;
  saveBlob(new Blob([problemsCsv(problems)], { type: 'text/csv' }), name);
}
