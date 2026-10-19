import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { WaterfallExample } from './WaterfallExample';

const LUBRICANTS = { inqToMqlRate: '2.00', mqlToDealRate: '5.00', averageDealSize: '10000.00' };

function inquiries() {
  return screen.getByRole('textbox', { name: 'Example number of inquiries' });
}

describe('WaterfallExample (FR-010, SC-006)', () => {
  it('works out the example for 250,000 inquiries, as goals are rounded', () => {
    renderWithProviders(<WaterfallExample {...LUBRICANTS} />);

    expect(screen.getByRole('heading', { name: 'Worked example' })).toBeInTheDocument();
    expect(screen.getByText('Calculated — not saved')).toBeInTheDocument();
    expect(inquiries()).toHaveValue('250,000');
    expect(
      screen.getByText('250,000 INQ → 5,000 MQL → 250 deals → 2,500,000.00 USD revenue'),
    ).toBeInTheDocument();
  });

  it('explains the formula in an ⓘ tip', () => {
    renderWithProviders(<WaterfallExample {...LUBRICANTS} />);

    expect(screen.getByRole('button', { name: 'About Worked example' })).toBeInTheDocument();
  });

  it('updates as soon as the example number of inquiries changes; nothing is saved', async () => {
    const { user } = renderWithProviders(<WaterfallExample {...LUBRICANTS} />);

    await user.clear(inquiries());
    await user.type(inquiries(), '25');

    // 25 × 2 % = 0.5 → 1 MQL (half up); 1 × 5 % = 0.05 → 0 deals.
    expect(screen.getByText('25 INQ → 1 MQL → 0 deals → 0.00 USD revenue')).toBeInTheDocument();

    await user.clear(inquiries());
    await user.type(inquiries(), '500');
    // 500 × 2 % = 10 MQL; 10 × 5 % = 0.5 → 1 deal.
    expect(
      screen.getByText('500 INQ → 10 MQL → 1 deal → 10,000.00 USD revenue'),
    ).toBeInTheDocument();
  });

  it('works with the rates and the deal size it is given (Mobility 1.0)', () => {
    renderWithProviders(
      <WaterfallExample inqToMqlRate="3" mqlToDealRate="4" averageDealSize="5000.00" />,
    );

    expect(
      screen.getByText('250,000 INQ → 7,500 MQL → 300 deals → 1,500,000.00 USD revenue'),
    ).toBeInTheDocument();
  });

  it('asks for both rates and the deal size when one is missing', () => {
    renderWithProviders(
      <WaterfallExample inqToMqlRate="2" mqlToDealRate={null} averageDealSize="10000.00" />,
    );

    expect(
      screen.getByText('Fill in both rates and the average deal size to see the example.'),
    ).toBeInTheDocument();
  });

  it('says what to type when the example is not a whole number from 0 to 1,000,000,000', async () => {
    const { user } = renderWithProviders(<WaterfallExample {...LUBRICANTS} />);

    await user.clear(inquiries());
    await user.type(inquiries(), '12.5');

    expect(inquiries()).toHaveAccessibleDescription(
      'Enter a whole number from 0 to 1,000,000,000.',
    );
    expect(screen.queryByText(/INQ →/)).not.toBeInTheDocument();
  });

  it('says when the revenue would be too large to calculate', () => {
    renderWithProviders(
      <WaterfallExample
        inqToMqlRate="100"
        mqlToDealRate="100"
        averageDealSize="100000000000000.00"
      />,
    );

    expect(screen.getByText('Too large to calculate — try fewer inquiries.')).toBeInTheDocument();
  });
});
