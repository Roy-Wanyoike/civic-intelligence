/**
 * Offline page — shown by the service worker when the network fails on a
 * navigation AND no cached copy of the requested page is available.
 *
 * The page is intentionally minimal: it explains what happened, lists the
 * things the user CAN do offline (read cached bills / acts / scenarios),
 * and offers a retry button. No client-side data fetching — the offline
 * page itself must be precached by the SW so it loads with zero network.
 */
import Link from 'next/link';

export const metadata = {
  title: 'Offline · Civic Intelligence',
  description: 'You are offline. Cached content is still available.',
  robots: { index: false, follow: false },
};

export default function OfflinePage() {
  return (
    <section className="mx-auto max-w-2xl px-4 py-16 text-center">
      <h1 className="text-3xl font-semibold text-forest">You are offline</h1>
      <p className="mt-4 text-base text-gray-700">
        We could not reach the Civic Intelligence server. Any page you have
        visited before is still readable from your device&rsquo;s cache. New
        pages and search will return when your connection does.
      </p>

      <button
        type="button"
        onClick={() => {
          if (typeof window !== 'undefined') window.location.reload();
        }}
        className="mt-8 inline-flex items-center rounded-md bg-forest px-4 py-2 text-sm font-medium text-white hover:bg-forest-dark focus:outline-none focus:ring-2 focus:ring-forest focus:ring-offset-2"
      >
        Try again
      </button>

      <h2 className="mt-12 text-sm font-semibold uppercase tracking-wide text-gray-500">
        Available offline
      </h2>
      <ul className="mt-4 space-y-2 text-left">
        <li>
          <Link href="/bills" className="text-forest underline-offset-2 hover:underline">
            Recently viewed Bills
          </Link>
        </li>
        <li>
          <Link href="/acts" className="text-forest underline-offset-2 hover:underline">
            Recently viewed Acts
          </Link>
        </li>
        <li>
          <Link href="/scenarios" className="text-forest underline-offset-2 hover:underline">
            Saved scenarios
          </Link>
        </li>
        <li>
          <Link href="/constitution" className="text-forest underline-offset-2 hover:underline">
            Constitution articles
          </Link>
        </li>
      </ul>

      <p className="mt-12 text-xs text-gray-400">
        Form submissions made while offline are saved on this device and sent
        automatically when you reconnect.
      </p>
    </section>
  );
}
