import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { z } from 'zod';
import { App } from '@/app/App';

// Zod never probes for eval, which the content security policy reports as a violation (038 FR-015).
z.config({ jitless: true });

const root = document.getElementById('root');
if (!root) {
  throw new Error('index.html has no element with id "root".');
}

createRoot(root).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
