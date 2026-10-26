import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { LevelTag } from './LevelTag';

describe('LevelTag', () => {
  it.each([
    ['plan', 'PLAN'],
    ['campaign', 'CAMPAIGN'],
    ['initiative', 'INITIATIVE'],
    ['activity', 'ACTIVITY'],
    ['lineItem', 'LINE ITEM'],
  ] as const)('writes the %s level as "%s" (design system §4)', (level, word) => {
    render(<LevelTag level={level} />);

    expect(screen.getByText(word)).toBeInTheDocument();
  });
});
