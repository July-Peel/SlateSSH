const CACHE = 'slatessh-shell-20260912-m3-6';
const SHELL = [
  '/',
  '/assets/css/app.css?v=20260912_m3_6',
  '/assets/js/app.js?v=20260912_m3_6',
  '/assets/js/pwa.js?v=20260912_m3_6',
  '/assets/vendor/material-symbols/rounded.css',
  '/assets/vendor/material-symbols/material-symbols-rounded.woff2',
  '/assets/vendor/alpine/alpine.min.js',
  '/assets/vendor/xterm/xterm.css',
  '/assets/vendor/xterm/xterm.js',
  '/assets/vendor/xterm/addon-fit.js',
  '/assets/vendor/xterm/addon-search.js',
  '/assets/vendor/guacamole/guacamole-common.min.js',
  '/assets/vendor/monaco/vs/loader.js',
  '/assets/icons/site.webmanifest',
  '/assets/icons/icon-192.png',
  '/assets/icons/icon-512.png',
  '/assets/icons/apple-touch-icon.png',
  '/assets/icons/favicon.svg'
];
const publicPaths = new Set(SHELL.filter(path => path !== '/'));

self.addEventListener('install', event => {
  event.waitUntil(caches.open(CACHE).then(cache => cache.addAll(SHELL)));
});
self.addEventListener('activate', event => {
  event.waitUntil((async () => {
    for (const key of await caches.keys()) {
      if (key.startsWith('slatessh-shell-') && key !== CACHE) await caches.delete(key);
    }
    await self.clients.claim();
  })());
});
self.addEventListener('message', event => {
  if (event.data?.type === 'SKIP_WAITING') self.skipWaiting();
});
self.addEventListener('fetch', event => {
  const request = event.request;
  const url = new URL(request.url);
  if (request.method !== 'GET' || url.origin !== self.location.origin) return;
  // Never intercept APIs, downloads, uploads, or WebSocket/tunnel requests.
  if (url.pathname.startsWith('/api/') || url.pathname === '/ws') return;
  if (request.mode === 'navigate') {
    // Keep the HTML and versioned scripts from the same installed release.
    event.respondWith(caches.open(CACHE).then(async cache => (await cache.match('/')) || fetch(request)));
  } else if (publicPaths.has(url.pathname + url.search)) {
    event.respondWith(caches.open(CACHE).then(async cache => (await cache.match(request)) || fetch(request)));
  }
});
