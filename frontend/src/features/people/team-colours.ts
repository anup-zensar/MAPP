import type { TeamColour } from './api';

/** The team palette (D-025), in the order of the Colour field. */
export const COLOURS: { id: TeamColour; label: string }[] = [
  { id: 'blue', label: 'Blue' },
  { id: 'violet', label: 'Violet' },
  { id: 'teal', label: 'Teal' },
  { id: 'magenta', label: 'Magenta' },
  { id: 'orange', label: 'Orange' },
  { id: 'green', label: 'Green' },
  { id: 'slate', label: 'Slate' },
  { id: 'gold', label: 'Gold' },
];
