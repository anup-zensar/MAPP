import { useState } from 'react';
import { TextField } from '@/ui/TextField/TextField';

/** The one field of the example side panel. */
export function PanelField() {
  const [name, setName] = useState('Helix launch');
  return (
    <TextField label="Name" name="panelName" marking="Required" value={name} onChange={setName} />
  );
}
