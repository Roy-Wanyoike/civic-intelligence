'use client';

/**
 * ServiceWorkerRegister — client component that registers /sw.js after the
 * app has hydrated, and handles the SW lifecycle:
 *
 *   - On first registration: logs success.
 *   - On update: prompts the user via window event so the app shell can
 *     surface a "Refresh to update" banner. The user can also trigger an
 *     immediate takeover via postMessage('SKIP_WAITING').
 *   - On error: logs but never throws — the app must keep working without
 *     a service worker.
 *
 * Mounted once from RootLayout (see src/app/layout.tsx).
 */
import { useEffect } from 'react';

export function ServiceWorkerRegister() {
  useEffect(() => {
    if (typeof window === 'undefined') return;
    if (!('serviceWorker' in navigator)) return;

    // Register after window load so it doesn't compete with first paint.
    const register = () => {
      navigator.serviceWorker
        .register('/sw.js', { scope: '/' })
        .then((reg) => {
          // Listen for updates — when a new SW takes over, emit a window
          // event the app shell can subscribe to.
          reg.addEventListener('updatefound', () => {
            const installing = reg.installing;
            if (!installing) return;
            installing.addEventListener('statechange', () => {
              if (
                installing.state === 'installed' &&
                navigator.serviceWorker.controller
              ) {
                window.dispatchEvent(
                  new CustomEvent('civic-sw-update-available')
                );
              }
            });
          });
        })
        .catch((err) => {
          // eslint-disable-next-line no-console
          console.warn('[sw] registration failed:', err);
        });
    };

    if (document.readyState === 'complete') {
      register();
    } else {
      window.addEventListener('load', register, { once: true });
    }

    // Let the user trigger an immediate takeover (e.g. from the "Refresh
    // to update" banner).
    const onSkipWaiting = () => {
      navigator.serviceWorker.getRegistration('/sw.js').then((reg) => {
        if (reg && reg.waiting) reg.waiting.postMessage('SKIP_WAITING');
      });
    };
    window.addEventListener('civic-sw-skip-waiting', onSkipWaiting);

    // Reload once the new controller has taken over.
    const onControllerChange = () => window.location.reload();
    navigator.serviceWorker.addEventListener('controllerchange', onControllerChange);

    return () => {
      window.removeEventListener('civic-sw-skip-waiting', onSkipWaiting);
      navigator.serviceWorker.removeEventListener(
        'controllerchange',
        onControllerChange
      );
    };
  }, []);

  return null;
}
