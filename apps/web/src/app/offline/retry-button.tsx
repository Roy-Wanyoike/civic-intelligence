'use client';

/**
 * OfflineRetryButton — the client half of the /offline page.
 *
 * The /offline page itself is a server component so the service worker can
 * precache it as a static asset. The retry button needs `onClick`, so it
 * lives here as a 'use client' island. The button is intentionally small:
 * on click, it forces a full-page reload (which the SW will then try to
 * serve from the network if it is back, or from cache if it is not).
 */
export function OfflineRetryButton() {
  return (
    <button
      type="button"
      onClick={() => {
        if (typeof window !== 'undefined') window.location.reload();
      }}
      className="mt-8 inline-flex items-center rounded-md bg-forest px-4 py-2 text-sm font-medium text-white hover:bg-forest-dark focus:outline-none focus:ring-2 focus:ring-forest focus:ring-offset-2"
    >
      Try again
    </button>
  );
}
