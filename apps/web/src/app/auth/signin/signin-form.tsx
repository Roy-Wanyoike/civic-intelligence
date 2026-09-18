'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Mail, Lock, Loader2, ArrowRight } from 'lucide-react';

/**
 * SignInForm — UI affordance for the /auth/signin page.
 *
 * No real authentication is performed. The form posts nothing to a backend;
 * the submit handler shows a brief "submitting" state and then redirects back
 * to the home page, mirroring the OIDC handoff that would happen in
 * production via Keycloak. The form fields exist purely so the page is a
 * recognisable sign-in surface.
 *
 * Tracking: ISSUE-108 (UX FAIL #26 — no sign-in affordance in the UI).
 */

export function SignInForm() {
  const router = useRouter();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);

  function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    // No backend call — Keycloak OIDC handles real auth in production.
    setLoading(true);
    setTimeout(() => {
      setLoading(false);
      router.push('/');
    }, 700);
  }

  return (
    <form className="mt-5 space-y-3" onSubmit={handleSubmit} aria-label="Sign in form">
      <div>
        <label htmlFor="email" className="sr-only">
          Email address
        </label>
        <div className="flex items-center gap-2 rounded-md border border-civic-border bg-civic-mist px-3 py-2.5 focus-within:border-civic-leaf">
          <Mail className="h-4 w-4 text-civic-stone" aria-hidden="true" />
          <input
            id="email"
            type="email"
            required
            autoComplete="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="you@example.com"
            className="flex-1 bg-transparent text-sm focus:outline-none"
          />
        </div>
      </div>
      <div>
        <label htmlFor="password" className="sr-only">
          Password
        </label>
        <div className="flex items-center gap-2 rounded-md border border-civic-border bg-civic-mist px-3 py-2.5 focus-within:border-civic-leaf">
          <Lock className="h-4 w-4 text-civic-stone" aria-hidden="true" />
          <input
            id="password"
            type="password"
            required
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="Password"
            className="flex-1 bg-transparent text-sm focus:outline-none"
          />
        </div>
      </div>

      <button
        type="submit"
        disabled={loading || !email || !password}
        className="mt-2 flex w-full items-center justify-center gap-2 rounded-md bg-civic-leaf px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-civic-leaf/90 disabled:cursor-not-allowed disabled:opacity-50"
      >
        {loading ? (
          <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
        ) : (
          <ArrowRight className="h-4 w-4" aria-hidden="true" />
        )}
        {loading ? 'Signing in…' : 'Sign in'}
      </button>
    </form>
  );
}
