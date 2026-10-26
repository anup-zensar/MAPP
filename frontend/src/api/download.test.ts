import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { mockApi } from '@/testing/mock-api';
import { reply } from '@/testing/replies';
import { api } from './client';
import { ApiError } from './errors';
import { downloadFromApi, fileNameFromDisposition, saveBlob } from './download';

const NAME = 'MAPP All work – Grid – 2026 – 18 Oct 2026.csv';
const DISPOSITION = `attachment; filename="MAPP All work - Grid - 2026 - 18 Oct 2026.csv"; filename*=UTF-8''${encodeURIComponent(NAME)}`;

let clicked: { href: string; download: string }[];

beforeEach(() => {
  clicked = [];
  // jsdom has no object URLs.
  URL.createObjectURL = () => 'blob:file-1';
  URL.revokeObjectURL = () => {};
  vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (
    this: HTMLAnchorElement,
  ) {
    clicked.push({ href: this.getAttribute('href') ?? '', download: this.download });
  });
});

afterEach(() => vi.restoreAllMocks());

describe('fileNameFromDisposition', () => {
  it('prefers the UTF-8 name, which keeps the dashes', () => {
    expect(fileNameFromDisposition(DISPOSITION)).toBe(NAME);
  });

  it('falls back to the plain name, quoted or not', () => {
    expect(fileNameFromDisposition('attachment; filename="report.xlsx"')).toBe('report.xlsx');
    expect(fileNameFromDisposition('attachment; filename=report.csv')).toBe('report.csv');
  });

  it('is null without a name', () => {
    expect(fileNameFromDisposition(null)).toBeNull();
    expect(fileNameFromDisposition('attachment')).toBeNull();
  });
});

describe('saveBlob', () => {
  it('lets the browser save the file under its name, and leaves no link behind', () => {
    saveBlob(new Blob(['a,b']), 'file.csv');

    expect(clicked).toEqual([{ href: 'blob:file-1', download: 'file.csv' }]);
    expect(document.querySelector('a[download]')).toBeNull();
  });
});

describe('downloadFromApi (FR-034)', () => {
  it('saves the answered file under the name the API gives and returns it', async () => {
    mockApi([
      reply(
        'GET /api/v1/all-work/export',
        () =>
          new Response('View,Grid', {
            headers: { 'Content-Type': 'text/csv', 'Content-Disposition': DISPOSITION },
          }),
      ),
    ]);

    const fileName = await downloadFromApi(
      api.GET('/api/v1/all-work/export', { params: { query: { format: 'csv' } }, parseAs: 'blob' }),
    );

    expect(fileName).toBe(NAME);
    expect(clicked).toEqual([{ href: 'blob:file-1', download: NAME }]);
  });

  it('turns an error answer into an ApiError with the API’s message, saving nothing', async () => {
    mockApi([
      reply('GET /api/v1/all-work/export', {
        status: 400,
        body: {
          error: {
            code: 'validation_failed',
            message: 'Choose a year from 2000 to 2100.',
            fields: [{ field: 'year', message: 'Choose a year from 2000 to 2100.' }],
          },
        },
      }),
    ]);

    const failure = await downloadFromApi(
      api.GET('/api/v1/all-work/export', {
        params: { query: { format: 'csv', year: 2026 } },
        parseAs: 'blob',
      }),
    ).catch((error: unknown) => error);

    expect(failure).toBeInstanceOf(ApiError);
    expect(failure).toHaveProperty('message', 'Choose a year from 2000 to 2100.');
    expect(clicked).toEqual([]);
  });
});
