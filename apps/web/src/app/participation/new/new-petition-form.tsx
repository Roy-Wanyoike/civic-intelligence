'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { createPetition, type Petition } from '@/lib/petitions-api';

// NewPetitionForm is the client-side form for POST /api/v1/petitions.
//
// On submit:
//   - Validates every required field is present + well-formed
//     (the backend re-validates, so this is purely UX).
//   - Disables the submit button + shows "Creating..." while the
//     request is in flight.
//   - On 201: navigates to the new petition's detail page so the
//     citizen can immediately sign their own petition (the first
//     signature on a freshly-created petition is a strong signal
//     to other citizens that the petition is real).
//   - On any other error: shows the error message + keeps the form
//     populated so the citizen can retry without re-typing.
//
// The closes_at field defaults to a date 90 days from today so the
// citizen does not have to think about a deadline unless they want
// to override it. The default is computed once at module load so
// re-renders don't drift the default forward.
const DEFAULT_CLOSES_AT = (() => {
  const d = new Date();
  d.setDate(d.getDate() + 90);
  // Strip milliseconds for RFC3339 compatibility.
  return d.toISOString().replace(/\.\d{3}Z$/, 'Z');
})();

export function NewPetitionForm() {
  const router = useRouter();
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [createdBy, setCreatedBy] = useState('');
  const [targetSignatures, setTargetSignatures] = useState('1000');
  const [closesAt, setClosesAt] = useState(DEFAULT_CLOSES_AT);
  const [country, setCountry] = useState('KE');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError(null);

    const trimmedTitle = title.trim();
    const trimmedDescription = description.trim();
    const trimmedCreatedBy = createdBy.trim();
    const trimmedCountry = country.trim().toUpperCase();
    const target = Number.parseInt(targetSignatures, 10);

    if (!trimmedTitle) {
      setError('Please enter a petition title.');
      return;
    }
    if (!trimmedDescription) {
      setError('Please enter a petition description.');
      return;
    }
    if (!trimmedCreatedBy) {
      setError('Please enter your name (the petitioner).');
      return;
    }
    if (!Number.isFinite(target) || target <= 0) {
      setError('Target signatures must be a positive integer.');
      return;
    }
    if (!trimmedCountry) {
      setError('Please enter a country code (e.g. KE).');
      return;
    }

    setLoading(true);
    try {
      const created: Petition = await createPetition({
        title: trimmedTitle,
        description: trimmedDescription,
        created_by: trimmedCreatedBy,
        target_signatures: target,
        closes_at: closesAt,
        country: trimmedCountry,
      });
      // Navigate to the new petition's detail page so the citizen
      // can immediately sign their own petition.
      router.push(`/participation/${created.id}`);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to create petition';
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const detail = (err as any)?.detail;
      const backendMessage =
        detail && typeof detail === 'object' && 'message' in detail
          ? String((detail as { message: unknown }).message)
          : message;
      setError(backendMessage);
    } finally {
      setLoading(false);
    }
  };

  return (
    <form
      onSubmit={handleSubmit}
      className="rounded-lg border border-stone-200 bg-white p-5"
    >
      <div className="grid gap-4">
        <label className="block text-sm">
          <span className="font-semibold text-stone-800">Title</span>
          <input
            type="text"
            name="title"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            required
            maxLength={240}
            className="mt-1 block w-full rounded-md border border-stone-300 bg-white px-3 py-2 text-sm text-stone-900 focus:border-civic-forest focus:outline-none focus:ring-1 focus:ring-civic-forest"
            placeholder="Petition to ..."
          />
        </label>

        <label className="block text-sm">
          <span className="font-semibold text-stone-800">Description</span>
          <textarea
            name="description"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            required
            rows={6}
            maxLength={8000}
            className="mt-1 block w-full rounded-md border border-stone-300 bg-white px-3 py-2 text-sm text-stone-900 focus:border-civic-forest focus:outline-none focus:ring-1 focus:ring-civic-forest"
            placeholder="Describe what you are asking the authority to do, and why. Cite the constitutional basis if applicable."
          />
        </label>

        <label className="block text-sm">
          <span className="font-semibold text-stone-800">Petitioner (your name + identifier)</span>
          <input
            type="text"
            name="created_by"
            value={createdBy}
            onChange={(e) => setCreatedBy(e.target.value)}
            required
            maxLength={160}
            className="mt-1 block w-full rounded-md border border-stone-300 bg-white px-3 py-2 text-sm text-stone-900 focus:border-civic-forest focus:outline-none focus:ring-1 focus:ring-civic-forest"
            placeholder="Wanjiku Mwangi (citizen, Nairobi County)"
            autoComplete="name"
          />
        </label>

        <div className="grid gap-4 sm:grid-cols-3">
          <label className="block text-sm">
            <span className="font-semibold text-stone-800">Target signatures</span>
            <input
              type="number"
              name="target_signatures"
              value={targetSignatures}
              onChange={(e) => setTargetSignatures(e.target.value)}
              required
              min={1}
              max={10_000_000}
              className="mt-1 block w-full rounded-md border border-stone-300 bg-white px-3 py-2 text-sm text-stone-900 focus:border-civic-forest focus:outline-none focus:ring-1 focus:ring-civic-forest"
            />
          </label>
          <label className="block text-sm">
            <span className="font-semibold text-stone-800">Country</span>
            <input
              type="text"
              name="country"
              value={country}
              onChange={(e) => setCountry(e.target.value)}
              required
              maxLength={2}
              className="mt-1 block w-full rounded-md border border-stone-300 bg-white px-3 py-2 text-sm text-stone-900 focus:border-civic-forest focus:outline-none focus:ring-1 focus:ring-civic-forest"
              placeholder="KE"
            />
          </label>
          <label className="block text-sm">
            <span className="font-semibold text-stone-800">Closes at</span>
            <input
              type="datetime-local"
              name="closes_at"
              // The backend expects RFC3339; datetime-local returns
              // a local-time string. We convert on submit via the
              // value the form holds — for the default we use a
              // pre-formatted RFC3339 string.
              value={toLocalInput(closesAt)}
              onChange={(e) => {
                // Convert the datetime-local value back to RFC3339.
                const v = e.target.value;
                if (!v) {
                  setClosesAt('');
                  return;
                }
                const d = new Date(v);
                setClosesAt(d.toISOString().replace(/\.\d{3}Z$/, 'Z'));
              }}
              required
              className="mt-1 block w-full rounded-md border border-stone-300 bg-white px-3 py-2 text-sm text-stone-900 focus:border-civic-forest focus:outline-none focus:ring-1 focus:ring-civic-forest"
            />
          </label>
        </div>

        {error && (
          <p className="text-sm text-rose-700" role="alert">
            {error}
          </p>
        )}

        <div className="flex items-center justify-between gap-3">
          <Link
            href="/participation"
            className="inline-flex items-center gap-1 text-sm text-civic-stone hover:text-civic-forest"
          >
            Cancel
          </Link>
          <button
            type="submit"
            disabled={loading}
            className="inline-flex items-center gap-2 rounded-lg bg-civic-forest px-4 py-2 text-sm font-semibold text-white transition hover:bg-civic-forest/90 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {loading ? 'Creating...' : 'Create petition'}
          </button>
        </div>
      </div>
    </form>
  );
}

// toLocalInput converts an RFC3339 string to the value expected by an
// <input type="datetime-local">. Returns '' if the input does not
// parse — the field renders empty rather than showing an invalid
// date string.
function toLocalInput(rfc3339: string): string {
  if (!rfc3339) return '';
  const d = new Date(rfc3339);
  if (Number.isNaN(d.getTime())) return '';
  // Format as YYYY-MM-DDTHH:mm in local time (the value format for
  // datetime-local inputs).
  const pad = (n: number) => String(n).padStart(2, '0');
  return (
    `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}` +
    `T${pad(d.getHours())}:${pad(d.getMinutes())}`
  );
}
