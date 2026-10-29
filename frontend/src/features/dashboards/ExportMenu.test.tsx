import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { ApiError } from '@/api/errors';
import { expectNoAxeViolations } from '@/testing/axe';
import { testUser } from '@/testing/fixtures/users';
import { findMessages, renderWithProviders } from '@/testing/render';
import { ExportMenu } from './ExportMenu';

describe('ExportMenu (036 US4, FR-024)', () => {
  it('offers PowerPoint and PDF and says which file it saved', async () => {
    const download = vi.fn(() =>
      Promise.resolve('SP&A One – Executive dashboard – 2026 – 28 Oct 2026.pptx'),
    );
    const { user } = renderWithProviders(<ExportMenu download={download} />, { user: testUser() });

    await user.click(screen.getByRole('button', { name: 'Export' }));
    expect((await screen.findAllByRole('menuitem')).map((item) => item.textContent)).toEqual([
      'PowerPoint (.pptx)',
      'PDF (.pdf)',
    ]);
    await expectNoAxeViolations();
    await user.click(screen.getByRole('menuitem', { name: 'PowerPoint (.pptx)' }));

    const messages = await findMessages();
    expect(
      within(messages).getByText(
        "Exported 'SP&A One – Executive dashboard – 2026 – 28 Oct 2026.pptx'",
      ),
    ).toBeInTheDocument();
    expect(download).toHaveBeenCalledWith('pptx');
  });

  it("shows the API's message when the file cannot be made", async () => {
    const download = vi.fn(() =>
      Promise.reject(
        new ApiError('forbidden', { message: "You can't export: ask an administrator." }),
      ),
    );
    const { user } = renderWithProviders(<ExportMenu download={download} />, { user: testUser() });

    await user.click(screen.getByRole('button', { name: 'Export' }));
    await user.click(await screen.findByRole('menuitem', { name: 'PDF (.pdf)' }));

    const messages = await findMessages();
    expect(
      within(messages).getByText("You can't export: ask an administrator."),
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Export' })).toBeInTheDocument();
  });
});
