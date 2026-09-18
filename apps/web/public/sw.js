/* eslint-disable no-restricted-globals */
/**
 * Civic Intelligence — Service Worker.
 *
 * Strategies:
 *   1. Static assets (CSS/JS/fonts/images)   → cache-first (fast, offline-capable)
 *   2. HTML navigations                       → network-first, fall back to cache, then to /offline
 *   3. Same-origin JSON API GETs              → stale-while-revalidate (fresh when online, cached when offline)
 *   4. POST/PUT/DELETE to /api/*              → background sync queue (replays when connection returns)
 *
 * Push notifications: a placeholder handler is registered. The actual
 * payload shape is owned by the notifications service; this SW only owns
 * the lifecycle (show / click / close).
 *
 * Versioning: bump CACHE_VERSION on every deploy that changes the precache
 * list or strategy. The `activate` event deletes old caches.
 */

const CACHE_VERSION = "v1.0.0";
const STATIC_CACHE = `civic-static-${CACHE_VERSION}`;
const RUNTIME_CACHE = `civic-runtime-${CACHE_VERSION}`;
const OFFLINE_URL = "/offline";

// Precache the offline page and the app shell on install. We DON'T precache
// the full Next.js bundle — that's what runtime caching is for.
const PRECACHE_URLS = [OFFLINE_URL, "/manifest.json"];

// Static-asset route matchers. Anything that looks like a build artifact or
// a font/image goes through cache-first.
const STATIC_ASSET_PATTERN =
  /\.(?:css|js|mjs|woff2?|ttf|eot|otf|png|jpg|jpeg|gif|webp|avif|svg|ico)$/i;
const STATIC_PATH_PREFIXES = ["/_next/static/", "/fonts/", "/icons/"];

// API routes that we cache (read-only). Mutations go through the background
// sync queue, never the cache.
const API_CACHE_PATTERN = /^\/api\/v1\/(bills|acts|search|scenarios|topics|gazette)\b/;

// ---------------------------------------------------------------------------
// Install — precache the app shell.
// ---------------------------------------------------------------------------
self.addEventListener("install", (event) => {
  event.waitUntil(
    (async () => {
      const cache = await caches.open(STATIC_CACHE);
      // Cache each precache URL individually so one failure doesn't abort
      // the whole install.
      await Promise.all(
        PRECACHE_URLS.map(async (url) => {
          try {
            await cache.add(url);
          } catch (err) {
            // eslint-disable-next-line no-console
            console.warn(`[sw] precache miss: ${url}`, err);
          }
        })
      );
      // Take control immediately so the SW is active on first navigation.
      await self.skipWaiting();
    })()
  );
});

// ---------------------------------------------------------------------------
// Activate — drop old caches and claim clients.
// ---------------------------------------------------------------------------
self.addEventListener("activate", (event) => {
  event.waitUntil(
    (async () => {
      const keys = await caches.keys();
      await Promise.all(
        keys
          .filter((k) => !k.endsWith(CACHE_VERSION))
          .map((k) => caches.delete(k))
      );
      await self.clients.claim();
    })()
  );
});

// ---------------------------------------------------------------------------
// Fetch — the strategy router.
// ---------------------------------------------------------------------------
self.addEventListener("fetch", (event) => {
  const req = event.request;

  // Only handle GETs here. Mutations go to the background-sync queue (see
  // the 'sync' listener below).
  if (req.method !== "GET") {
    if (req.method === "POST" && new URL(req.url).pathname.startsWith("/api/")) {
      event.respondWith(queueMutation(req));
    }
    return;
  }

  // Skip non-http(s) requests (chrome-extension://, data:, blob:).
  const url = new URL(req.url);
  if (url.protocol !== "http:" && url.protocol !== "https:") return;

  // Skip cross-origin requests — they have their own CORS / caching policies.
  if (url.origin !== self.location.origin) return;

  // 1. Navigations (HTML pages) → network-first with offline fallback.
  if (req.mode === "navigate") {
    event.respondWith(networkFirstForHTML(req));
    return;
  }

  // 2. Static assets → cache-first.
  if (isStaticAsset(url)) {
    event.respondWith(cacheFirst(req, STATIC_CACHE));
    return;
  }

  // 3. Read-only API GETs → stale-while-revalidate.
  if (API_CACHE_PATTERN.test(url.pathname)) {
    event.respondWith(staleWhileRevalidate(req, RUNTIME_CACHE));
    return;
  }

  // 4. Everything else — try network, fall back to runtime cache, then bail.
  event.respondWith(
    fetch(req).catch(async () => {
      const cached = await caches.match(req);
      return cached || Response.error();
    })
  );
});

// ---------------------------------------------------------------------------
// Strategies
// ---------------------------------------------------------------------------

async function cacheFirst(req, cacheName) {
  const cached = await caches.match(req);
  if (cached) return cached;
  try {
    const res = await fetch(req);
    if (res && res.ok) await putInCache(req, res, cacheName);
    return res;
  } catch (err) {
    return Response.error();
  }
}

async function networkFirstForHTML(req) {
  try {
    const res = await fetch(req);
    if (res && res.ok) {
      const cache = await caches.open(RUNTIME_CACHE);
      cache.put(req, res.clone());
    }
    return res;
  } catch (err) {
    // Network failed — try cache, then fall back to the offline page.
    const cached = await caches.match(req);
    if (cached) return cached;
    const offline = await caches.match(OFFLINE_URL);
    return offline || (await caches.match("/"));
  }
}

async function staleWhileRevalidate(req, cacheName) {
  const cache = await caches.open(cacheName);
  const cached = await cache.match(req);
  const network = fetch(req)
    .then((res) => {
      if (res && res.ok) cache.put(req, res.clone());
      return res;
    })
    .catch(() => cached);
  return cached || network;
}

async function putInCache(req, res, cacheName) {
  const cache = await caches.open(cacheName);
  await cache.put(req, res.clone());
}

function isStaticAsset(url) {
  if (STATIC_ASSET_PATTERN.test(url.pathname)) return true;
  return STATIC_PATH_PREFIXES.some((p) => url.pathname.startsWith(p));
}

// ---------------------------------------------------------------------------
// Background sync — replay queued mutations when the connection returns.
// ---------------------------------------------------------------------------
const BG_SYNC_QUEUE = "civic-mutation-queue";

async function queueMutation(req) {
  try {
    const cache = await caches.open(BG_SYNC_QUEUE);
    // We can't store a Request directly in the cache (it's consumed once).
    // Snapshot it as a plain object.
    const body = await req.clone().text();
    const snapshot = new Response(
      JSON.stringify({
        url: req.url,
        method: req.method,
        headers: Object.fromEntries(req.headers.entries()),
        body,
      }),
      { headers: { "Content-Type": "application/json" } }
    );
    const key = new Request(`__bg_sync__/${Date.now()}-${Math.random()}`);
    await cache.put(key, snapshot);
    // Register for background sync (best-effort; not all browsers support it).
    if ("sync" in self.registration) {
      try {
        await self.registration.sync.register("civic-mutation-replay");
      } catch (err) {
        // eslint-disable-next-line no-console
        console.warn("[sw] sync.register failed", err);
      }
    }
    return new Response(
      JSON.stringify({ queued: true, message: "Saved offline — will retry when back online." }),
      { status: 202, headers: { "Content-Type": "application/json" } }
    );
  } catch (err) {
    return new Response(JSON.stringify({ queued: false, error: String(err) }), {
      status: 503,
      headers: { "Content-Type": "application/json" },
    });
  }
}

self.addEventListener("sync", (event) => {
  if (event.tag === "civic-mutation-replay") {
    event.waitUntil(replayQueuedMutations());
  }
});

async function replayQueuedMutations() {
  const cache = await caches.open(BG_SYNC_QUEUE);
  const keys = await cache.keys();
  for (const key of keys) {
    if (!key.url.includes("__bg_sync__")) continue;
    const res = await cache.match(key);
    const snap = JSON.parse(await res.text());
    try {
      const replayRes = await fetch(snap.url, {
        method: snap.method,
        headers: snap.headers,
        body: snap.body || undefined,
      });
      if (replayRes.ok) {
        await cache.delete(key);
      }
      // If not ok, leave in queue for next sync event.
    } catch (err) {
      // eslint-disable-next-line no-console
      console.warn("[sw] replay failed (will retry):", snap.url, err);
      break; // stop on first network failure; remaining will retry later
    }
  }
}

// ---------------------------------------------------------------------------
// Push notifications — placeholder handler.
// ---------------------------------------------------------------------------
self.addEventListener("push", (event) => {
  let payload = { title: "Civic Intelligence", body: "You have a new update." };
  try {
    if (event.data) payload = { ...payload, ...event.data.json() };
  } catch (err) {
    if (event.data) payload.body = event.data.text();
  }
  event.waitUntil(
    self.registration.showNotification(payload.title, {
      body: payload.body,
      icon: payload.icon || "/icons/192.png",
      badge: payload.badge || "/icons/badge-72.png",
      tag: payload.tag || "civic-update",
      data: payload.data || {},
    })
  );
});

self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  const target = event.notification.data?.url || "/";
  event.waitUntil(
    (async () => {
      const allClients = await self.clients.matchAll({
        type: "window",
        includeUncontrolled: true,
      });
      for (const client of allClients) {
        if (client.url.includes(target) && "focus" in client) {
          return client.focus();
        }
      }
      if (self.clients.openWindow) return self.clients.openWindow(target);
    })()
  );
});

// ---------------------------------------------------------------------------
// Message channel — lets the app ask the SW to update immediately.
// ---------------------------------------------------------------------------
self.addEventListener("message", (event) => {
  if (event.data === "SKIP_WAITING") self.skipWaiting();
});
