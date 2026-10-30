import { screen, waitFor } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it, vi } from 'vitest';
import { expectNoAxeViolations } from '@/testing/axe';
import { renderWithProviders } from '@/testing/render';
import { ConfirmDialog } from '../ConfirmDialog/ConfirmDialog';
import { FormDialog } from '../FormDialog/FormDialog';
import { FormErrorSummary } from '../FormErrorSummary/FormErrorSummary';
import { FormPanel } from '../FormPanel/FormPanel';
import type { FieldMarking } from '../FieldShell/FieldShell';
import { TextAreaField } from '../TextAreaField/TextAreaField';
import { TextField } from '../TextField/TextField';

const ignore = () => {};

/** The legends shown on screen: "* Required". */
function legends(): Element[] {
  return [...document.querySelectorAll('[data-required-legend]:not([hidden])')];
}

function Panel({ marking }: { marking?: FieldMarking }) {
  return (
    <FormPanel
      title="New team"
      isDirty={false}
      saving={false}
      problems={[]}
      failure={null}
      onSave={ignore}
      onClose={ignore}
    >
      <TextField label="Name" name="name" value="" onChange={ignore} marking={marking} />
    </FormPanel>
  );
}

describe('RequiredLegend (038 FR-064, D-364)', () => {
  it('038 FR-064: a side panel with a required field shows "* Required" once, before its first field', async () => {
    renderWithProviders(<Panel marking="Required" />);

    await waitFor(() => expect(legends()).toHaveLength(1));
    const [legend] = legends();
    expect(legend).toHaveTextContent('* Required');
    // Screen readers hear "required" with each field instead (038 FR-063).
    expect(legend).toHaveAttribute('aria-hidden', 'true');
    const name = screen.getByRole('textbox', { name: 'Name' });
    expect(legend?.compareDocumentPosition(name)).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
  });

  it.each<FieldMarking | undefined>(['Needed to publish', undefined])(
    '038 FR-064: no legend when no field is required to save (%s)',
    (marking) => {
      renderWithProviders(<Panel marking={marking} />);
      expect(legends()).toHaveLength(0);
    },
  );

  it('038 FR-064: a dialog whose form also has an error summary still shows the legend once', async () => {
    renderWithProviders(
      <FormDialog
        open
        onOpenChange={ignore}
        title="Rename view"
        submitLabel="Rename"
        onSubmit={ignore}
      >
        <FormErrorSummary errors={[]} />
        <TextField label="Name" name="name" value="" onChange={ignore} marking="Required" />
      </FormDialog>,
    );

    await waitFor(() => expect(legends()).toHaveLength(1));
  });

  it('038 FR-064: the legend appears when a required field appears later', async () => {
    function Growing() {
      const [more, setMore] = useState(false);
      return (
        <form aria-label="Plan">
          <FormErrorSummary errors={[]} />
          <button type="button" onClick={() => setMore(true)}>
            More
          </button>
          {more && (
            <TextField label="Name" name="name" value="" onChange={ignore} marking="Required" />
          )}
        </form>
      );
    }
    const { user } = renderWithProviders(<Growing />);
    expect(legends()).toHaveLength(0);

    await user.click(screen.getByRole('button', { name: 'More' }));
    await waitFor(() => expect(legends()).toHaveLength(1));
  });

  it('038 FR-064: a question with a required reason shows the legend, and axe finds nothing', async () => {
    renderWithProviders(
      <ConfirmDialog
        open
        onOpenChange={ignore}
        title="Decline 'Launch film'?"
        confirmLabel="Decline"
        onConfirm={vi.fn()}
      >
        <TextAreaField label="Reason" name="reason" value="" onChange={ignore} marking="Required" />
      </ConfirmDialog>,
    );

    await waitFor(() => expect(legends()).toHaveLength(1));
    await expectNoAxeViolations(document.body);
  });
});
