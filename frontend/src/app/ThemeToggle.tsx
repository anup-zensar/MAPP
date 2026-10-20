import { Moon, Sun } from 'lucide-react';
import { Button, Tooltip } from '@/components';
import { useSession } from './useSession';

/**
 * The light/dark switch of the top bar (FR-014): it offers the other mode — a moon while the app
 * is light, a sun while it is dark. Signed in, the choice is saved for the person; signed out,
 * only in this browser (research R16).
 */
export function ThemeToggle() {
  const { theme, updatePreferences } = useSession();
  const offered = theme === 'dark' ? 'light' : 'dark';
  const name = offered === 'dark' ? 'Switch to dark mode' : 'Switch to light mode';

  return (
    <Tooltip content={name}>
      <Button
        variant="tertiary"
        icon={offered === 'dark' ? <Moon /> : <Sun />}
        aria-label={name}
        onClick={() => updatePreferences({ theme: offered })}
      />
    </Tooltip>
  );
}
