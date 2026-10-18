import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
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
