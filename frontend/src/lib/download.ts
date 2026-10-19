/**
 * Saving a file that the API answers (feature 010's export; feature 012's templates and error
 * reports use it too). The browser saves it under the name the API gives in Content-Disposition.
 *
 *   const fileName = await downloadFromApi(
 *     api.GET('/api/all-work/export', { params: { query }, parseAs: 'blob' }),
 *   );
 */
import { call } from '@/api/client';

/** What openapi-fetch gives back for a call answered as a file (parseAs: 'blob'). */
interface FileResult {
  data?: Blob;
  error?: unknown;
  response: Response;
}

/**
 * Waits for the file, saves it and returns its name. A failure throws an ApiError, exactly as
 * call() does, so the page shows the API's plain-language message.
 */
export async function downloadFromApi(
  request: Promise<FileResult>,
  fallbackName = 'download',
): Promise<string> {
  let disposition: string | null = null;
  const blob = await call(
    request.then((result) => {
      disposition = result.response.headers.get('Content-Disposition');
      return result;
    }),
  );
  const fileName = fileNameFromDisposition(disposition) ?? fallbackName;
  saveBlob(blob, fileName);
  return fileName;
}

/**
 * The file name of a Content-Disposition header: the UTF-8 one (filename*) when there is one, as
 * it keeps characters such as "–"; else the plain one; null when there is none.
 */
export function fileNameFromDisposition(header: string | null): string | null {
  if (!header) {
    return null;
  }
  const encoded = /filename\*\s*=\s*utf-8''([^;]+)/i.exec(header);
  if (encoded) {
    try {
      return decodeURIComponent(encoded[1].trim());
    } catch {
      // A badly encoded name: the plain one below will do.
    }
  }
  const plain = /filename\s*=\s*(?:"([^"]*)"|([^;]+))/i.exec(header);
  const name = plain ? (plain[1] ?? plain[2]).trim() : '';
  return name === '' ? null : name;
}

/** Lets the browser save `blob` as a file called `fileName`. */
export function saveBlob(blob: Blob, fileName: string): void {
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = fileName;
  link.style.display = 'none';
  document.body.append(link);
  link.click();
  link.remove();
  // The browser has taken the file; the address can go once this click is handled.
  window.setTimeout(() => URL.revokeObjectURL(url), 0);
}
