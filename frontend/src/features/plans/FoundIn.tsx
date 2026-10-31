import type { PlanFoundIn } from './api';
import styles from './FoundIn.module.css';

const LEVEL_WORDS: Record<PlanFoundIn['match']['level'], string> = {
  campaign: 'campaign',
  initiative: 'initiative',
  activity: 'activity',
  lineItem: 'line item',
};

function foundInText({ match, more }: PlanFoundIn): string {
  const first = `Found in ${LEVEL_WORDS[match.level]} '${match.name}' (${match.id})`;
  return more > 0 ? `${first} and ${more} more` : first;
}

/** Why the search lists a plan whose own name and ID do not hold the text. */
export function FoundIn({ foundIn }: { foundIn?: PlanFoundIn }) {
  return foundIn ? <p className={styles.foundIn}>{foundInText(foundIn)}</p> : null;
}
