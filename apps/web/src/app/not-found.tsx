import Link from 'next/link';

export default function NotFound() {
  return (
    <div className="mx-auto max-w-md px-4 py-24 text-center">
      <p className="font-serif text-6xl font-bold text-civic-leaf">404</p>
      <h1 className="mt-4 font-serif text-2xl font-semibold text-civic-forest">Page not found</h1>
      <p className="mt-2 text-sm text-civic-stone">
        The page you are looking for does not exist or has been moved.
      </p>
      <Link
        href="/"
        className="mt-6 inline-flex items-center gap-2 rounded-md bg-civic-leaf px-5 py-2.5 font-semibold text-white hover:bg-civic-leaf/90"
      >
        Return home
      </Link>
    </div>
  );
}
