import { useState } from 'react';
import { Card } from '@/ui/Card/Card';
import { PersonPicker } from '@/ui/PersonPicker/PersonPicker';
import { TeamPicker } from '@/ui/TeamPicker/TeamPicker';
import { PEOPLE, TEAMS } from './gallery-data';
import styles from './ComponentGalleryPage.module.css';

export function PickersExample() {
  const [owner, setOwner] = useState('U-1002');
  const [teams, setTeams] = useState(['T-1001', 'T-1003']);
  return (
    <Card title="PersonPicker and TeamPicker">
      <div className={styles.fields}>
        <TeamPicker
          label="Teams"
          name="galleryTeams"
          teams={TEAMS}
          value={teams}
          onChange={setTeams}
        />
        <PersonPicker
          label="Owner"
          name="galleryOwner"
          marking="Needed to publish"
          people={PEOPLE}
          currentUserId="U-1002"
          teamIds={teams}
          value={owner}
          onChange={setOwner}
        />
      </div>
    </Card>
  );
}
