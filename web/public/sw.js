const CACHE_VERSION = 'nexus-shell-v2';
const SHELL = [
  '/',
  '/manifest.webmanifest',
  '/nexus-icon-192.png',
  '/nexus-icon-512.png',
  '/nexus-icon.png',
];

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches
      .open(CACHE_VERSION)
      .then((cache) => cache.addAll(SHELL))
      .then(() => self.skipWaiting())
      .catch(() => undefined),
  );
});

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((keys) =>
        Promise.all(keys.filter((key) => key !== CACHE_VERSION).map((key) => caches.delete(key))),
      )
      .then(() => self.clients.claim()),
  );
});

self.addEventListener('fetch', (event) => {
  const request = event.request;
  if (request.method !== 'GET') return;

  const url = new URL(request.url);
  if (url.origin !== self.location.origin) return;
  if (url.pathname.startsWith('/api/')) return;

  if (request.mode === 'navigate') {
    event.respondWith(
      fetch(request).catch(() =>
        caches.match('/').then((cached) => cached || caches.match(request)),
      ),
    );
    return;
  }

  event.respondWith(
    caches.match(request).then(
      (cached) =>
        cached ||
        fetch(request).then((response) => {
          if (response && response.ok) {
            const copy = response.clone();
            void caches.open(CACHE_VERSION).then((cache) => cache.put(request, copy));
          }
          return response;
        }),
    ),
  );
});

// Web Push arrives with an aes128gcm payload already decrypted by the browser.
// The notification is always same-origin; nothing from the payload is executed.
self.addEventListener('push', (event) => {
  let data = { title: 'Nexus', body: '', url: '/', tag: 'nexus' };
  if (event.data) {
    try {
      const parsed = event.data.json();
      if (parsed && typeof parsed === 'object') data = { ...data, ...parsed };
    } catch {
      data.body = event.data.text();
    }
  }
  const url = typeof data.url === 'string' && data.url.startsWith('/') ? data.url : '/';
  event.waitUntil(
    self.registration.showNotification(String(data.title || 'Nexus'), {
      body: String(data.body || ''),
      icon: '/nexus-icon-192.png',
      badge: '/nexus-icon-96.png',
      tag: String(data.tag || 'nexus'),
      data: { url },
      renotify: Boolean(data.renotify),
    }),
  );
});

// Focus an existing Nexus tab when possible instead of opening another one.
self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  const target = new URL(event.notification.data?.url || '/', self.location.origin).href;
  event.waitUntil(
    self.clients
      .matchAll({ type: 'window', includeUncontrolled: true })
      .then((clients) => {
        for (const client of clients) {
          if (client.url === target && 'focus' in client) return client.focus();
        }
        return self.clients.openWindow ? self.clients.openWindow(target) : undefined;
      })
      .catch(() => undefined),
  );
});
