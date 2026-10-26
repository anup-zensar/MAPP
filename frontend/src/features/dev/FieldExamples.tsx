import { useState } from 'react';
import { PasswordField } from '@/ui/PasswordField/PasswordField';
import { TextField } from '@/ui/TextField/TextField';
import styles from './ComponentGalleryPage.module.css';

/** The text fields: with a marking and an error, with a hint, and a password. */
export function FieldExamples() {
  const [campaignName, setCampaignName] = useState('');
  const [budgetOwner, setBudgetOwner] = useState('Priya Raman');
  const [password, setPassword] = useState('');

  return (
    <div className={styles.fields}>
      <TextField
        label="Campaign name"
        name="campaignName"
        marking="Required"
        value={campaignName}
        onChange={setCampaignName}
        error="Enter a name."
      />
      <TextField
        label="Budget owner"
        name="budgetOwner"
        marking="Needed to publish"
        hint="The person who answers for the budget."
        value={budgetOwner}
        onChange={setBudgetOwner}
      />
      <PasswordField label="Password" name="password" value={password} onChange={setPassword} />
    </div>
  );
}
