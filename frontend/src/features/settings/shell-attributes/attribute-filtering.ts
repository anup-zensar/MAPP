import type { ShellAttribute } from './api';
import type { HelpFilter, StatusFilter } from './schemas';

/** True when the attribute has no help text, so its ⓘ is missing (016 FR-048). */
export function lacksHelp(attribute: ShellAttribute): boolean {
  return attribute.helpText.trim() === '';
}

export function hasHelpState(attribute: ShellAttribute, help: HelpFilter): boolean {
  return help === 'all' || lacksHelp(attribute);
}

export function hasStatus(attribute: ShellAttribute, status: StatusFilter): boolean {
  return status === 'all' || attribute.active === (status === 'active');
}

export function matches(attribute: ShellAttribute, q: string): boolean {
  const text = q.trim().toLowerCase();
  return (
    text === '' ||
    attribute.label.toLowerCase().includes(text) ||
    attribute.section.toLowerCase().includes(text)
  );
}

export interface AttributeSection {
  name: string;
  items: ShellAttribute[];
}

/** The attributes in display order, as one group per section (data model §4.2). */
export function bySection(attributes: ShellAttribute[]): AttributeSection[] {
  const sections: AttributeSection[] = [];
  for (const attribute of attributes) {
    const last = sections[sections.length - 1];
    if (last && last.name === attribute.section) {
      last.items.push(attribute);
    } else {
      sections.push({ name: attribute.section, items: [attribute] });
    }
  }
  return sections;
}
