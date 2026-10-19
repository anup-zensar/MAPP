import { afterEach, describe, expect, it, vi } from 'vitest';
import { downloadOptionsTemplate, optionsTemplate } from './importTemplate';

describe('optionsTemplate (FR-031)', () => {
  it('names the file after the attribute, with the header "Option"', () => {
    expect(optionsTemplate({ label: 'Cost centre', parentAttribute: null })).toEqual({
      fileName: 'Cost centre – options template.csv',
      text: 'Option\r\n',
    });
  });

  it('adds the "Parent option" column for an attribute that depends on another', () => {
    expect(optionsTemplate({ label: 'Region', parentAttribute: { name: 'Cluster' } })).toEqual({
      fileName: 'Region – options template.csv',
      text: 'Option,Parent option\r\n',
    });
  });

  it('leaves out characters a file name cannot hold', () => {
    expect(optionsTemplate({ label: 'Country / OU', parentAttribute: null }).fileName).toBe(
      'Country - OU – options template.csv',
    );
  });
});

describe('downloadOptionsTemplate', () => {
  afterEach(() => {
    Reflect.deleteProperty(URL, 'createObjectURL');
    Reflect.deleteProperty(URL, 'revokeObjectURL');
  });

  it('saves the template under its name, as UTF-8 that Excel reads', async () => {
    let saved: Blob | undefined;
    Object.defineProperty(URL, 'createObjectURL', {
      configurable: true,
      value: (blob: Blob) => {
        saved = blob;
        return 'blob:template';
      },
    });
    Object.defineProperty(URL, 'revokeObjectURL', { configurable: true, value: vi.fn() });
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (
      this: HTMLAnchorElement,
    ) {
      expect(this.download).toBe('Cost centre – options template.csv');
      expect(this.href).toBe('blob:template');
    });

    downloadOptionsTemplate({ label: 'Cost centre', parentAttribute: null });

    expect(click).toHaveBeenCalledTimes(1);
    expect(saved?.type).toBe('text/csv;charset=utf-8');
    // The byte-order mark (3 bytes in UTF-8), then the header row.
    expect(saved?.size).toBe(3 + 'Option\r\n'.length);
    expect(document.querySelector('a[download]')).toBeNull();
    click.mockRestore();
  });
});
