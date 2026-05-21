const CACHE_VERSION = 'v2';
const STATIC_CACHE = `waffle-${CACHE_VERSION}-static`;
const PAGES_CACHE = `waffle-${CACHE_VERSION}-pages`;

const STATIC_ASSETS = [
  '/static/css/output.css',
  '/static/favicon.svg',
  '/static/img/logo.png',
  '/static/manifest.json',
  '/static/offline.html',
  '/static/js/sw.js',
  '/static/js/offline-handler.js',
  '/static/js/spot-selection.js',
  '/static/js/websocket-client.js',
  '/static/js/reports.js',
  '/static/js/admin-spot-actions.js',
];

const PUBLIC_PAGES = [
  '/',
  '/waffles',
];

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(STATIC_CACHE)
      .then((cache) => cache.addAll(STATIC_ASSETS))
      .then(() => caches.open(PAGES_CACHE))
      .then((cache) => cache.addAll(PUBLIC_PAGES))
      .then(() => self.skipWaiting())
  );
});

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((cacheNames) => {
      return Promise.all(
        cacheNames.map((cacheName) => {
          if (cacheName.startsWith('waffle-') && cacheName !== STATIC_CACHE && cacheName !== PAGES_CACHE) {
            return caches.delete(cacheName);
          }
        })
      );
    }).then(() => self.clients.claim())
  );
});

function isNetworkOnly(request) {
  const url = new URL(request.url);

  if (request.method !== 'GET') {
    return true;
  }

  if (url.pathname.startsWith('/api/') ||
      url.pathname.startsWith('/ws/') ||
      url.pathname === '/health') {
    return true;
  }

  if (url.pathname.startsWith('/admin/')) {
    return true;
  }

  return false;
}

function isStaticAsset(request) {
  const url = new URL(request.url);
  return url.pathname.startsWith('/static/');
}

function isPublicPage(request) {
  const url = new URL(request.url);
  return url.pathname === '/' ||
         url.pathname === '/waffles' ||
         url.pathname.startsWith('/waffle/') ||
         url.pathname.startsWith('/buyer/');
}

self.addEventListener('fetch', (event) => {
  const request = event.request;

  if (isNetworkOnly(request)) {
    return;
  }

  if (isStaticAsset(request)) {
    event.respondWith(
      caches.match(request).then((cached) => {
        if (cached) {
          return cached;
        }
        return fetch(request).then((response) => {
          if (!response || response.status !== 200 || response.type !== 'basic') {
            return response;
          }
          const responseClone = response.clone();
          caches.open(STATIC_CACHE).then((cache) => {
            cache.put(request, responseClone);
          });
          return response;
        });
      })
    );
    return;
  }

  if (isPublicPage(request)) {
    event.respondWith(
      caches.match(request).then((cached) => {
        const fetchPromise = fetch(request).then((response) => {
          if (response && response.status === 200) {
            const responseClone = response.clone();
            caches.open(PAGES_CACHE).then((cache) => {
              cache.put(request, responseClone);
            });
          }
          return response;
        }).catch(() => {
          return caches.match('/static/offline.html');
        });

        return cached || fetchPromise;
      })
    );
    return;
  }
});

self.addEventListener('message', (event) => {
  if (event.data && event.data.type === 'SKIP_WAITING') {
    self.skipWaiting();
  }
});
