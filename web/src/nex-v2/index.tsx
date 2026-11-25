import React from 'react';
import { createRoot } from 'react-dom/client';
import '../app/workspace-styles/01-foundation.css';
import '../styles/globals.scss';
import './i18n';
import { NexAvatarView } from './NexAvatarView';
import { NexPresenceApp } from './NexPresenceApp';

if (typeof document !== 'undefined') {
  document.documentElement.dataset.nativeHud = 'true';
  document.body.dataset.nativeHud = 'true';
}

const params = typeof window !== 'undefined' ? new URLSearchParams(window.location.search) : null;
const labMode = params?.get('lab') === '1';

const container = document.getElementById('root');
if (container) {
  createRoot(container).render(labMode ? <NexAvatarView labMode /> : <NexPresenceApp />);
}
