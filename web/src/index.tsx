import React from 'react';
import { createRoot } from 'react-dom/client';
import { App } from './App';
import './i18n';
import './styles/globals.scss';
import './app/workspace-os.css';
import { initPlatformBridge } from './platform';

const platformBridge = initPlatformBridge();
if (typeof document !== 'undefined') {
  document.documentElement.dataset.nexusPlatform = platformBridge.kind;
}

const container = document.getElementById('root');
if (container) {
  const root = createRoot(container);
  root.render(<App />);
}

if (typeof navigator !== 'undefined' && 'serviceWorker' in navigator) {
  window.addEventListener('load', () => {
    navigator.serviceWorker.register('/sw.js').catch(() => undefined);
  });
}
