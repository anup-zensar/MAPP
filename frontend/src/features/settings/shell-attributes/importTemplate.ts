/**
 * The option template of an attribute (feature 002, US6, FR-031): a CSV file built in the browser
 * with the header row only — "Option", plus "Parent option" for an attribute that depends on
 * another — named "<label> – options template.csv".
 */

/** What the template needs to know about the attribute. */
export interface TemplateAttribute {
  label: string;
  /** The attribute it depends on; null when none. */
  parentAttribute: { name: string } | null;
}

/** The template's file name and its text. */
export interface OptionsTemplate {
  fileName: string;
  text: string;
}

/** Characters a file name can't hold on Windows and macOS. */
const NOT_IN_FILE_NAMES = /[\\/:*?"<>|]/g;

/** "Cost centre – options template.csv" with the header "Option" (or "Option,Parent option"). */
export function optionsTemplate(attribute: TemplateAttribute): OptionsTemplate {
  const name = attribute.label.replace(NOT_IN_FILE_NAMES, '-');
  return {
    fileName: `${name} – options template.csv`,
    text: attribute.parentAttribute ? 'Option,Parent option\r\n' : 'Option\r\n',
  };
}

/**
 * Saves the template on the person's computer. The file starts with a byte-order mark so that
 * Excel reads its UTF-8 text correctly; the import ignores it.
 */
export function downloadOptionsTemplate(attribute: TemplateAttribute): void {
  const { fileName, text } = optionsTemplate(attribute);
  const url = URL.createObjectURL(new Blob(['﻿', text], { type: 'text/csv;charset=utf-8' }));
  const link = document.createElement('a');
  link.href = url;
  link.download = fileName;
  document.body.append(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}
