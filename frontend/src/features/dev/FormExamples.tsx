import { zodResolver } from '@hookform/resolvers/zod';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { Button } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { FormDialog } from '@/ui/FormDialog/FormDialog';
import { FormField } from '@/ui/FormField/FormField';
import { FormFooter } from '@/ui/FormFooter/FormFooter';
import { FormPanel } from '@/ui/FormPanel/FormPanel';
import { FormSection } from '@/ui/FormSection/FormSection';
import { TextField } from '@/ui/TextField/TextField';
import { CHECK_ON_SAVE } from '@/ui/useFormProblems';
import styles from './ComponentGalleryPage.module.css';
import { copyNameSchema } from './schemas';
import type { CopyNameValues } from './schemas';

export function FormExamples() {
  const [dialogOpen, setDialogOpen] = useState(false);
  const [panelOpen, setPanelOpen] = useState(false);
  const form = useForm<CopyNameValues>({
    resolver: zodResolver(copyNameSchema),
    defaultValues: { copyName: 'FY2026 Lubricants Europe (copy)' },
    ...CHECK_ON_SAVE,
  });
  return (
    <Card title="FormSection, FormFooter, FormDialog and FormPanel">
      <div className={styles.fields}>
        <FormSection title="Basics" description="Sections of a full-page form, in order.">
          <p>Fields go here.</p>
        </FormSection>
        <FormFooter>
          <Button variant="primary" onClick={() => setDialogOpen(true)}>
            Open form dialog
          </Button>
          <Button onClick={() => setPanelOpen(true)}>Open form panel</Button>
        </FormFooter>
      </div>
      <FormDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        title="Copy plan 'FY2026 Lubricants Europe'"
        submitLabel="Copy plan"
        onSubmit={() => void form.handleSubmit(() => setDialogOpen(false))()}
      >
        <FormField
          control={form.control}
          name="copyName"
          render={(field) => (
            <TextField label="Name of the copy" {...field} name="galleryCopyName" />
          )}
        />
      </FormDialog>
      {panelOpen && (
        <FormPanel
          title="New team"
          isDirty={false}
          saving={false}
          problems={[]}
          failure={null}
          onSave={() => setPanelOpen(false)}
          onClose={() => setPanelOpen(false)}
        >
          <p>The fields of a short record.</p>
        </FormPanel>
      )}
    </Card>
  );
}
