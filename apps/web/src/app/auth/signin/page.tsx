import type { Metadata } from 'next';
import Link from 'next/link';
import { Scale, ArrowLeft, ShieldCheck, AlertTriangle } from 'lucide-react';
import { RealityBadge } from '@/components/reality-labels';
import { SignInForm } from './signin-form';

export const metadata: Metadata = {
  title: 'Sign in',
  description:
    'Sign in to Civic Intelligence to ask questions, follow Bills, and run research missions.',
};

export const dynamic = 'force-dynamic';

export default function SignInPage() {
  return (
    <div className="mx-auto flex max-w-md flex-col px-4 py-12 sm:px-6">
      <Link
        href="/"
        className="mb-6 inline-flex items-center gap-1 text-sm text-civic-stone hover:text-civic-leaf"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden="true" /> Back to home
      </Link>

      <div className="rounded-2xl border border-civic-border bg-civic-paper p-6 shadow-sm sm:p-8">
        <div className="flex items-center gap-2">
          <Scale className="h-7 w-7 text-civic-forest" aria-hidden="true" />
          <span className="font-serif text-xl font-semibold text-civic-forest">
            Civic Intelligence
          </span>
        </div>
        <h1 className="mt-4 font-serif text-2xl font-semibold text-civic-forest">
          Sign in
        </h1>
        <p className="mt-1 text-sm text-civic-stone">
          Sign in to ask questions, follow Bills, run research missions, and
          receive notifications.
        </p>

        {/* OIDC notice — production auth is handled by Keycloak, not this
            form. The form below is a UI affordance only (issue #108). */}
        <div
          className="mt-4 rounded-lg border border-civic-border bg-civic-mist p-3"
          role="note"
        >
          <div className="flex items-start gap-2">
            <ShieldCheck className="mt-0.5 h-4 w-4 flex-shrink-0 text-civic-leaf" aria-hidden="true" />
            <p className="text-xs text-civic-stone">
              <strong className="text-civic-ink">Authentication is handled via Keycloak OIDC.</strong>{' '}
              This form will redirect to the Keycloak login page in production.
              The form below is a UI affordance and does not authenticate
              against a real user store.
            </p>
          </div>
        </div>

        <SignInForm />

        {/* Google placeholder */}
        <div className="mt-6">
          <div className="relative">
            <div className="absolute inset-0 flex items-center" aria-hidden="true">
              <div className="w-full border-t border-civic-border" />
            </div>
            <div className="relative flex justify-center">
              <span className="bg-civic-paper px-2 text-xs text-civic-stone">
                or
              </span>
            </div>
          </div>
          <Link
            href="/"
            className="mt-4 flex w-full items-center justify-center gap-2 rounded-md border border-civic-border bg-civic-paper px-4 py-2.5 text-sm font-medium text-civic-ink hover:border-civic-leaf hover:text-civic-leaf"
          >
            <GoogleIcon />
            Sign in with Google
          </Link>
          <p className="mt-2 text-center text-xs text-civic-stone">
            <RealityBadge kind="LIMITATION" />
            <span className="ml-2">
              Placeholder — wires to Keycloak social login when configured.
            </span>
          </p>
        </div>

        {/* Footer */}
        <p className="mt-6 text-center text-xs text-civic-stone">
          Don&apos;t have an account?{' '}
          <Link href="/about" className="text-civic-leaf hover:underline">
            Learn how to get access.
          </Link>
        </p>
      </div>

      <p className="mt-6 flex items-center justify-center gap-1 text-xs text-civic-stone">
        <AlertTriangle className="h-3 w-3" aria-hidden="true" />
        No real credentials are transmitted by this page.
      </p>
    </div>
  );
}

function GoogleIcon() {
  return (
    <svg
      aria-hidden="true"
      width="16"
      height="16"
      viewBox="0 0 18 18"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        fill="#4285F4"
        d="M17.64 9.2c0-.637-.057-1.251-.164-1.84H9v3.481h4.844c-.209 1.125-.843 2.078-1.796 2.717v2.258h2.908c1.702-1.567 2.684-3.875 2.684-6.615z"
      />
      <path
        fill="#34A853"
        d="M9 18c2.43 0 4.467-.806 5.956-2.18l-2.908-2.259c-.806.54-1.836.86-3.048.86-2.344 0-4.328-1.584-5.036-3.711H.957v2.332C2.438 15.983 5.482 18 9 18z"
      />
      <path
        fill="#FBBC05"
        d="M3.964 10.71c-.18-.54-.282-1.117-.282-1.71s.102-1.17.282-1.71V4.958H.957C.347 6.173 0 7.548 0 9s.347 2.827.957 4.042l3.007-2.332z"
      />
      <path
        fill="#EA4335"
        d="M9 3.58c1.321 0 2.508.454 3.44 1.345l2.582-2.58C13.464.891 11.427 0 9 0 5.482 0 2.438 2.017.957 4.958L3.964 7.29C4.672 5.163 6.656 3.58 9 3.58z"
      />
    </svg>
  );
}
