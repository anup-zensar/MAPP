import { useState } from 'react';
import { Card } from '@/ui/Card/Card';
import { CheckboxField } from '@/ui/CheckboxField/CheckboxField';
import { TextAreaField } from '@/ui/TextAreaField/TextAreaField';
import { TextField } from '@/ui/TextField/TextField';
import styles from './ComponentGalleryPage.module.css';

export function TextFieldsExample() {
  const [description, setDescription] = useState('');
  const [aop, setAop] = useState(false);
  const [budgetName, setBudgetName] = useState('');
  return (
    <Card title="TextAreaField, CheckboxField and suggestions">
      <div className={styles.fields}>
        <TextAreaField
          label="Description"
          name="galleryDescription"
          maxLength={60}
          hint="Counts down in its last 50 characters."
          value={description}
          onChange={setDescription}
        />
        <CheckboxField
          label="Strategic / AOP plan"
          name="galleryAop"
          checked={aop}
          onChange={setAop}
        />
        <CheckboxField
          label="Ongoing (no end date)"
          name="galleryOngoing"
          checked
          onChange={() => {}}
          disabledReason="Ongoing can't be undone — to plan with an end date, create a new activity."
        />
        <TextField
          label="Budget name"
          name="galleryBudgetName"
          hint="Items with the same year and budget name form one finance budget."
          suggestions={['Europe – Retail', 'Asia – B2B']}
          value={budgetName}
          onChange={setBudgetName}
        />
      </div>
    </Card>
  );
}
