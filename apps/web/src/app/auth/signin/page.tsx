import type { Metadata } from 'next';
import Link from 'next/link';
import { Scale, ArrowLeft, ShieldCheck, AlertTriangle, FileText, Globe, BookOpen, Radio } from 'lucide-react';
import { RealityBadge } from '@/components/reality-labels';
import { SignInForm } from './signin-form';

export const metadata: Metadata = {
  title: 'Sign in · Civic Intelligence',
  description:
    'Sign in to Civic Intelligence to ask questions, follow Bills, and run research missions across 14 African countries.',
};

export const dynamic = 'force-dynamic';

export default function SignInPage() {
  return (
    <div className="min-h-screen lg:grid lg:grid-cols-2">
      {/* === LEFT SECTION: Branding + Marketing === */}
      <div className="relative hidden overflow-hidden lg:flex lg:flex-col lg:justify-between">
        {/* Background image — Kenya Parliament */}
        <div
          className="absolute inset-0 bg-cover bg-center"
          style={{ backgroundImage: 'url(/hero-parliament.jpg)' }}
          aria-hidden="true"
        />
        {/* Gradient overlay */}
        <div
          className="absolute inset-0 bg-gradient-to-br from-civic-forest/95 via-civic-forest/85 to-civic-ink/95"
          aria-hidden="true"
        />

        {/* Top: Brand + back link */}
        <div className="relative z-10 p-10">
          <Link
            href="/"
            className="inline-flex items-center gap-2 text-sm text-white/70 transition hover:text-white"
          >
            <ArrowLeft className="h-4 w-4" aria-hidden="true" />
            Back to home
          </Link>
          <div className="mt-8 flex items-center gap-3">
            <Scale className="h-8 w-8 text-civic-acacia" aria-hidden="true" />
            <span className="font-serif text-2xl font-bold text-white">
              Civic Intelligence
            </span>
          </div>
        </div>

        {/* Center: Marketing copy */}
        <div className="relative z-10 flex-1 flex flex-col justify-center px-10">
          <h2 className="max-w-md font-serif text-3xl font-bold leading-tight text-white">
            Understand what your government is doing.
          </h2>
          <p className="mt-4 max-w-md text-lg text-white/80">
            Evidence-grounded explanations of legislation, regulations, and civic activity
            across 14 African countries. No legal jargon. No political bias.
          </p>

          {/* Feature bullets */}
          <div className="mt-8 space-y-3">
            {[
              { icon: FileText, text: 'Track Bills through every legislative stage' },
              { icon: Globe, text: 'Compare civic indicators across 14 countries' },
              { icon: BookOpen, text: 'Read the Hansard in plain language' },
              { icon: Radio, text: 'Daily gazette alerts by keyword' },
            ].map((feature, i) => (
              <div key={i} className="flex items-center gap-3">
                <feature.icon className="h-5 w-5 flex-shrink-0 text-civic-acacia" aria-hidden="true" />
                <span className="text-sm text-white/85">{feature.text}</span>
              </div>
            ))}
          </div>
        </div>

        {/* Bottom: Stats */}
        <div className="relative z-10 p-10">
          <div className="flex gap-6 text-sm">
            <div>
              <div className="font-serif text-2xl font-bold text-white">50+</div>
              <div className="text-xs text-white/60">Bills tracked</div>
            </div>
            <div>
              <div className="font-serif text-2xl font-bold text-white">14</div>
              <div className="text-xs text-white/60">Countries</div>
            </div>
            <div>
              <div className="font-serif text-2xl font-bold text-white">1,093</div>
              <div className="text-xs text-white/60">Tests passing</div>
            </div>
          </div>
        </div>
      </div>

      {/* === RIGHT SECTION: Auth form === */}
      <div className="flex min-h-screen flex-col bg-civic-paper px-4 py-12 sm:px-6 lg:min-h-0 lg:justify-center">
        {/* Mobile: brand + back link */}
        <div className="lg:hidden">
          <Link
            href="/"
            className="mb-6 inline-flex items-center gap-1 text-sm text-civic-stone hover:text-civic-leaf"
          >
            <ArrowLeft className="h-4 w-4" aria-hidden="true" /> Back to home
          </Link>
          <div className="mb-6 flex items-center gap-2">
            <Scale className="h-7 w-7 text-civic-forest" aria-hidden="true" />
            <span className="font-serif text-xl font-semibold text-civic-forest">
              Civic Intelligence
            </span>
          </div>
        </div>

        {/* Form card */}
        <div className="mx-auto w-full max-w-md">
          <h1 className="font-serif text-3xl font-bold text-civic-forest">
            Welcome back
          </h1>
          <p className="mt-2 text-sm text-civic-stone">
            Sign in to ask questions, follow Bills, run research missions, and
            receive notifications.
          </p>

          {/* OIDC notice */}
          <div
            className="mt-5 rounded-lg border border-civic-border bg-civic-mist p-3"
            role="note"
          >
            <div className="flex items-start gap-2">
              <ShieldCheck className="mt-0.5 h-4 w-4 flex-shrink-0 text-civic-leaf" aria-hidden="true" />
              <p className="text-xs text-civic-stone">
                <strong className="text-civic-ink">Authentication is handled via Keycloak OIDC.</strong>{' '}
                This form will redirect to the Keycloak login page in production.
              </p>
            </div>
          </div>

          {/* Sign in form */}
          <div className="mt-6">
            <SignInForm />
          </div>

          {/* Divider */}
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

            {/* Google sign-in */}
            <Link
              href="/"
              className="mt-4 flex w-full items-center justify-center gap-2 rounded-md border border-civic-border bg-civic-paper px-4 py-2.5 text-sm font-medium text-civic-ink transition hover:border-civic-leaf hover:text-civic-leaf"
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

          {/* Sign up link */}
          <div className="mt-8 rounded-lg border border-civic-border bg-civic-mist/50 p-4 text-center">
            <p className="text-sm text-civic-stone">
              Don&apos;t have an account?
            </p>
            <p className="mt-1 text-xs text-civic-stone">
              Civic Intelligence is free for citizens.{' '}
              <Link href="/about" className="font-medium text-civic-leaf hover:underline">
                Learn how to get access →
              </Link>
            </p>
          </div>

          {/* Footer */}
          <p className="mt-6 flex items-center justify-center gap-1 text-xs text-civic-stone">
            <AlertTriangle className="h-3 w-3" aria-hidden="true" />
            No real credentials are transmitted by this page.
          </p>
        </div>
      </div>
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
