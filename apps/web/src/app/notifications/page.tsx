import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Notifications',
  description: 'Your civic notifications.',
};

export default function NotificationsPage() {
  return (
    <div className="mx-auto max-w-3xl px-4 py-10 sm:px-6">
      <h1 className="font-serif text-3xl font-semibold text-civic-forest">Notifications</h1>
      <p className="mt-2 text-sm text-civic-stone">Verified civic alerts for Bills, committees, and topics you follow.</p>
      <div className="mt-8 rounded-lg border border-dashed border-civic-border p-8 text-center text-sm text-civic-stone">
        No notifications yet. Follow a Bill to receive alerts when it changes.
      </div>
    </div>
  );
}
