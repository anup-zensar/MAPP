import { Link } from 'react-router';
import { Card } from '@/components';
import { BuildingPage } from '@/features/placeholders/BuildingPage';
import { SETTINGS_SECTIONS } from './sections';

/**
 * Settings (feature 002, FR-001) — STUB of the shared foundation: feature 002 replaces this file
 * with the page of section cards (sections.ts, counts from GET /api/settings/overview and each
 * section's ActiveCount). Until then it links to every section so the stubs can be reached.
 */
export function SettingsPage() {
  return (
    <BuildingPage
      eyebrow="Settings"
      title="Settings"
      description="Set up the lists and rules every plan uses. Changes here apply to everyone straight away."
    >
      <Card title="Sections">
        <ul>
          {SETTINGS_SECTIONS.map((section) => (
            <li key={section.path}>
              <Link to={section.path}>{section.title}</Link>
            </li>
          ))}
        </ul>
      </Card>
    </BuildingPage>
  );
}
