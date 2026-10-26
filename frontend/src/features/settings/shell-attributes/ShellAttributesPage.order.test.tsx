import { screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { findMessages } from '@/testing/render';
import {
  BRANDS,
  BUSINESS_UNIT_LOCK,
  MEDIA_CHANNEL,
  OLD_NOTES,
  page,
  PIR,
} from '@/testing/fixtures/shell-attributes';
import { renderShellAttributesPage, rowNames } from '@/testing/fixtures/shell-attribute-pages';
import { reply } from '@/testing/replies';

describe('ShellAttributesPage: order (FR-019, SC-009)', () => {
  /** The level's new order after moving PIR above Media channel, as the API answers it. */
  const PIR_FIRST = page([
    { ...PIR, position: 1, version: 2 },
    { ...MEDIA_CHANNEL, position: 2, version: 2 },
    BRANDS,
    OLD_NOTES,
  ]);

  it('moves an attribute with the keyboard, keeps the focus on its button and says so', async () => {
    const { user, api } = renderShellAttributesPage(
      '/settings/shell-attributes?level=activity&status=all',
      [
        reply(
          'POST /api/v1/shell-attributes/{attributeId}/move',
          { body: PIR_FIRST },
          { attributeId: 'SA-1011' },
        ),
      ],
    );

    const up = await screen.findByRole('button', { name: "Move 'PIR' up" });
    up.focus();
    await user.keyboard('{Enter}');

    await waitFor(() =>
      expect(rowNames()).toEqual([
        'Shell attributes',
        'PIR',
        'Media channel',
        'Lubricants attributes',
        'Brands',
        'Old notes',
      ]),
    );
    expect(api.bodiesOf('POST /api/v1/shell-attributes/SA-1011/move')).toEqual([
      { what: 'attribute', direction: 'up', version: 1 },
    ]);
    await waitFor(() =>
      expect(screen.getByRole('button', { name: "Move 'PIR' up" })).toHaveFocus(),
    );
    expect(screen.getByText("Moved 'PIR' up.")).toHaveAttribute('role', 'status');
  });

  it('keeps edge buttons focusable and says why they do nothing', async () => {
    const { user, api } = renderShellAttributesPage(
      '/settings/shell-attributes?level=activity&status=all',
    );

    const up = await screen.findByRole('button', { name: "Move 'Media channel' up" });
    expect(up).toHaveAttribute('aria-disabled', 'true');
    await user.hover(up);
    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      "'Media channel' is already first in its section.",
    );
    await user.click(up);
    expect(screen.getByRole('button', { name: "Move 'Old notes' down" })).toHaveAttribute(
      'aria-disabled',
      'true',
    );
    expect(
      screen.getByRole('button', { name: "Move section 'Shell attributes' up" }),
    ).toHaveAttribute('aria-disabled', 'true');
    expect(api.requests.filter((request) => request.method === 'POST')).toEqual([]);
  });

  it('explains the order buttons while searching or filtering', async () => {
    const { user, api } = renderShellAttributesPage('/settings/shell-attributes?level=activity');

    const down = await screen.findByRole('button', { name: "Move 'Media channel' down" });
    expect(down).toHaveAttribute('aria-disabled', 'true');
    // Keyboard focus, which shows the tooltip: Tab from the button just before it.
    screen.getByRole('button', { name: "Move 'Media channel' up" }).focus();
    await user.tab();
    expect(down).toHaveFocus();
    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      'Clear the search and filters to change the order.',
    );
    await user.keyboard('{Enter}');
    expect(api.requests.filter((request) => request.method === 'POST')).toEqual([]);
  });

  it('moves a whole section', async () => {
    const { user, api } = renderShellAttributesPage(
      '/settings/shell-attributes?level=activity&status=all',
      [
        reply(
          'POST /api/v1/shell-attributes/{attributeId}/move',
          {
            body: page([BRANDS, OLD_NOTES, MEDIA_CHANNEL, PIR]),
          },
          { attributeId: 'SA-1012' },
        ),
      ],
    );

    await user.click(
      await screen.findByRole('button', { name: "Move section 'Lubricants attributes' up" }),
    );

    await waitFor(() => expect(rowNames()[0]).toBe('Lubricants attributes'));
    expect(api.bodiesOf('POST /api/v1/shell-attributes/SA-1012/move')).toEqual([
      { what: 'section', direction: 'up', version: 1 },
    ]);
    expect(screen.getByText("Moved 'Lubricants attributes' up.")).toHaveAttribute('role', 'status');
  });

  it('shows why the API refused a move', async () => {
    const { user } = renderShellAttributesPage(
      '/settings/shell-attributes?level=activity&status=all',
      [
        reply(
          'POST /api/v1/shell-attributes/{attributeId}/move',
          {
            status: 400,
            body: {
              error: {
                code: 'validation_failed',
                message: 'Check the move.',
                fields: [{ field: 'direction', message: 'Old notes must stay below Brands.' }],
              },
            },
          },
          { attributeId: 'SA-1013' },
        ),
      ],
    );

    await user.click(await screen.findByRole('button', { name: "Move 'Old notes' up" }));

    expect(await findMessages()).toHaveTextContent('Old notes must stay below Brands.');
  });

  it('shows the lock of Business unit on its order buttons (FR-023)', async () => {
    const { user } = renderShellAttributesPage('/settings/shell-attributes');

    const down = await screen.findByRole('button', { name: "Move 'Business unit' down" });
    expect(down).toHaveAttribute('aria-disabled', 'true');
    // Keyboard focus, which shows the tooltip: Tab from the button just before it.
    screen.getByRole('button', { name: "Move 'Business unit' up" }).focus();
    await user.tab();
    expect(down).toHaveFocus();
    expect(await screen.findByRole('tooltip')).toHaveTextContent(BUSINESS_UNIT_LOCK);
  });
});
