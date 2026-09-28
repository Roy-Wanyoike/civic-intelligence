'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { signPetition, type Petition, type PetitionSignature } from '@/lib/petitions-api';

interface SignFormProps {
  petition: Petition;
}

// SignForm is the client-side form for POST /api/v1/petitions/{id}/sign.
//
// On submit:
//   - Validates name + email are non-empty + the email is well-formed
//     (the backend re-validates, so this is purely UX).
//   - Disables the submit button + shows "Signing..." while the
//     request is in flight.
//   - On 201: shows a success banner with the created signature +
//     the updated signatures_count so the citizen can see their
//     signature has been recorded (verified: false — the email
//     verification pipeline will confirm the signer's email out-of-band).
//   - On 409: shows a "petition closed" banner — the petition is not
//     in a state that accepts signatures (closed / answered).
//   - On any other error: shows the error message + keeps the form
//     populated so the citizen can retry without re-typing.
//
// The form deliberately does NOT collect anything beyond name + email
// — the platform's public-participation portal keeps the sign
// friction minimal so citizens can sign in < 30 seconds.
export function SignForm({ petition: initialPetition }: SignFormProps) {
  const router = useRouter();
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<{ signature: PetitionSignature; petition: Petition } | null>(null);

  const isOpen = initialPetition.status === 'open';

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError(null);
    setSuccess(null);

    const trimmedName = name.trim();
    const trimmedEmail = email.trim();
    if (!trimmedName) {
      setError('Please enter your name.');
      return;
    }
    if (!trimmedEmail || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(trimmedEmail)) {
      setError('Please enter a valid email address.');
      return;
    }

    setLoading(true);
    try {
      const resp = await signPetition(initialPetition.id, {
        name: trimmedName,
        email: trimmedEmail,
      });
      setSuccess(resp);
      setName('');
      setEmail('');
      // Refresh the server-rendered petition list so the new count is
      // visible if the citizen navigates back to the list view.
      router.refresh();
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to sign petition';
      // The ApiError carries a `detail` field for the JSON body —
      // surface the backend's `message` if present.
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

  if (!isOpen) {
    return (
      <div className="rounded-lg border border-stone-200 bg-stone-50 p-4 text-sm text-stone-700">
        This petition is <strong>{initialPetition.status}</strong> and is no
        longer accepting signatures.
      </div>
    );
  }

  if (success) {
    return (
      <div className="rounded-lg border border-emerald-300 bg-emerald-50 p-4 text-sm text-emerald-900">
        <p className="font-semibold">Signature recorded.</p>
        <p className="mt-1">
          Thank you, {success.signature.name}. Your signature has been added
          to the petition — the signature count is now{' '}
          <strong>{success.petition.signatures_count.toLocaleString('en-KE')}</strong>.
        </p>
        <p className="mt-2 text-xs text-emerald-800">
          A verification email will be sent to {success.signature.email} once
          the email pipeline ships. Until then, your signature is recorded
          with <code>verified: false</code>.
        </p>
        <button
          type="button"
          onClick={() => setSuccess(null)}
          className="mt-3 inline-flex items-center gap-1 text-xs font-semibold text-emerald-900 hover:underline"
        >
          Sign again on behalf of someone else
        </button>
      </div>
    );
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="rounded-lg border border-stone-200 bg-white p-5"
    >
      <h2 className="font-serif text-lg font-semibold text-civic-forest">
        Sign this petition
      </h2>
      <p className="mt-1 text-xs text-civic-stone">
        Your name + email are collected so the platform can verify your
        identity out-of-band. The email is NOT published on the public
        signatures list.
      </p>

      <div className="mt-4 grid gap-3 sm:grid-cols-2">
        <label className="block text-sm">
          <span className="font-semibold text-stone-800">Name</span>
          <input
            type="text"
            name="name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
            maxLength={120}
            className="mt-1 block w-full rounded-md border border-stone-300 bg-white px-3 py-2 text-sm text-stone-900 focus:border-civic-forest focus:outline-none focus:ring-1 focus:ring-civic-forest"
            placeholder="Wanjiku Mwangi"
            autoComplete="name"
          />
        </label>
        <label className="block text-sm">
          <span className="font-semibold text-stone-800">Email</span>
          <input
            type="email"
            name="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            maxLength={160}
            className="mt-1 block w-full rounded-md border border-stone-300 bg-white px-3 py-2 text-sm text-stone-900 focus:border-civic-forest focus:outline-none focus:ring-1 focus:ring-civic-forest"
            placeholder="wanjiku.mwangi@example.org"
            autoComplete="email"
          />
        </label>
      </div>

      {error && (
        <p className="mt-3 text-sm text-rose-700" role="alert">
          {error}
        </p>
      )}

      <button
        type="submit"
        disabled={loading}
        className="mt-4 inline-flex items-center gap-2 rounded-lg bg-civic-forest px-4 py-2 text-sm font-semibold text-white transition hover:bg-civic-forest/90 disabled:cursor-not-allowed disabled:opacity-50"
      >
        {loading ? 'Signing...' : 'Sign this petition'}
      </button>
    </form>
  );
}
