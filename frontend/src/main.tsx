import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
// Jost 600 and 700 (Latin), the open-licence heading face (SIL OFL 1.1, see README.md), served
// with the app itself: no request leaves the computer for a font.
import '@fontsource/jost/latin-600.css';
import '@fontsource/jost/latin-700.css';
import '@/styles/tokens.css';
import '@/styles/global.css';
import { App } from '@/app/App';

const root = document.getElementById('root');
if (!root) {
  throw new Error('index.html has no element with id "root".');
}

createRoot(root).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
