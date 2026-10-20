import type { RouteObject } from 'react-router';
import { ImportRatesPage } from './currencies/ImportRatesPage';
import { CURRENCY_IMPORT_PATH } from './currencies/currencyPaths';
import { FieldHelpPage } from './field-help/FieldHelpPage';
import { FIELD_HELP_PATH } from './field-help/settingsSections';

/**
 * The Settings pages feature 016 adds (plan "Pages"), for `src/app/router.tsx` to add to the Plan
 * module's routes (`...SETTINGS_016_ROUTES`): Import GSAP rates and Field help. Each page names
 * itself for the browser tab.
 */
export const SETTINGS_016_ROUTES: RouteObject[] = [
  {
    path: CURRENCY_IMPORT_PATH,
    element: <ImportRatesPage />,
    handle: { title: 'Import GSAP rates' },
  },
  { path: FIELD_HELP_PATH, element: <FieldHelpPage />, handle: { title: 'Field help' } },
];
