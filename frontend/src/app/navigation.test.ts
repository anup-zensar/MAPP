import { describe, expect, it } from 'vitest';
import { COMING_SOON_SECTIONS } from './navigation';

describe('Sections of the Plan module', () => {
  it('are all built: none is Coming soon since 010 and 011 (FR-001, FR-014)', () => {
    expect(COMING_SOON_SECTIONS).toEqual([]);
  });
});
