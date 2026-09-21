'use client';

import { useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import {
  AlertCircle,
  CheckCircle2,
  ExternalLink,
  Loader2,
  Send,
} from 'lucide-react';
import { cn } from '@/lib/utils';

/**
 * Correction categories — mirror trust.corrections.category CHECK constraint
 * in migration 018_trust_schema.up.sql. Each category carries a short helper
 * that explains when to use it; shown to the user in the dropdown.
 */
const CATEGORIES: { value: string; label: string; hint: string }[] = [
  {
    value: 'wrong_fact',
    label: 'Wrong fact',
    hint: 'A factual claim on the page is incorrect (e.g. wrong date, wrong amount).',
  },
  {
    value: 'wrong_source',
    label: 'Wrong source',
    hint: 'The page attributes a claim to the wrong official source.',
  },
  {
    value: 'wrong_citation',
    label: 'Wrong citation',
    hint: 'The citation points to the wrong document, page, or section.',
  },
  {
    value: 'outdated',
    label: 'Outdated',
    hint: 'The information was correct when published but is now out of date.',
  },
  {
    value: 'incorrect_interpretation',
    label: 'Incorrect interpretation',
    hint: 'The facts are right but the plain-language explanation is misleading.',
  },
  {
    value: 'missing_information',
    label: 'Missing information',
    hint: 'Important context the page omits that should be added.',
  },
  {
    value: 'conflicting_sources',
    label: 'Conflicting sources',
    hint: 'Two official sources disagree and the page only shows one.',
  },
  {
    value: 'broken_document',
    label: 'Broken document',
    hint: 'A link to a PDF / source document is broken or returns 404.',
  },
];

type SubmitState =
  | { kind: 'idle' }
  | { kind: 'submitting' }
  | { kind: 'success'; id: string }
  | { kind: 'error'; message: string };

export function ReportForm() {
  const router = useRouter();
  const [state, setState] = useState<SubmitState>({ kind: 'idle' });
  const [category, setCategory] = useState<string>('');
  const [pageUrl, setPageUrl] = useState('');
  const [description, setDescription] = useState('');
  const [evidenceUrl, setEvidenceUrl] = useState('');
  const [email, setEmail] = useState('');
  const [targetType, setTargetType] = useState<string>('bill');
  const [errors, setErrors] = useState<Record<string, string>>({});

  // Client-side validation mirrors the server-side rules in
  // services/api/cmd/corrections.go::CorrectionStore.Submit so the user gets
  // instant feedback instead of a 400 round-trip.
  function validate(): Record<string, string> {
    const e: Record<string, string> = {};
    if (!category) e.category = 'Please pick a category.';
    if (!pageUrl.trim()) {
      e.pageUrl = 'Page URL is required.';
    } else if (!/^https?:\/\/.+/.test(pageUrl.trim())) {
      e.pageUrl = 'Page URL must start with http:// or https://';
    }
    if (!description.trim()) {
      e.description = 'Please describe what is wrong.';
    } else if (description.trim().length < 20) {
      e.description = 'Description must be at least 20 characters.';
    }
    if (evidenceUrl.trim() && !/^https?:\/\/.+/.test(evidenceUrl.trim())) {
      e.evidence = 'Evidence URL must start with http:// or https://';
    }
    if (!email.trim()) {
      // Anonymous submissions require an email so reviewers can follow up.
      e.email = 'Email is required so we can follow up.';
    } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim())) {
      e.email = 'Please enter a valid email address.';
    }
    return e;
  }

  async function onSubmit(ev: React.FormEvent<HTMLFormElement>) {
    ev.preventDefault();
    const e = validate();
    setErrors(e);
    if (Object.keys(e).length > 0) return;

    setState({ kind: 'submitting' });

    // Derive target_id from the page URL when possible. For civic-intelligence
    // URLs of the form /bills/{id}, /acts/{id}, etc., the last path segment
    // is the canonical ID. This is best-effort — the server stores whatever
    // the user submits and the review queue picks it up.
    const segments = pageUrl.split('/').filter(Boolean);
    const lastSegment = segments[segments.length - 1] ?? pageUrl;
    const detectedType = (() => {
      for (const seg of segments) {
        if (['bills', 'bill'].includes(seg)) return 'bill';
        if (['acts', 'act'].includes(seg)) return 'act';
        if (['regulations', 'regulation'].includes(seg)) return 'regulation';
        if (['policies', 'policy'].includes(seg)) return 'policy';
        if (['people', 'person'].includes(seg)) return 'person';
        if (['committees', 'committee'].includes(seg)) return 'committee';
      }
      return targetType;
    })();
    if (detectedType !== targetType) setTargetType(detectedType);

    const body = {
      target_type: detectedType,
      target_id: lastSegment,
      category,
      reason: description.trim(),
      page_url: pageUrl.trim(),
      evidence: evidenceUrl.trim(),
      submitted_email: email.trim(),
    };

    try {
      const resp = await fetch('/api/v1/corrections', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      if (!resp.ok) {
        let detail: { error?: string; message?: string } | null = null;
        try { detail = await resp.json(); } catch { /* ignore */ }
        const msg =
          detail?.message ||
          detail?.error ||
          `Submission failed (HTTP ${resp.status}).`;
        setState({ kind: 'error', message: msg });
        return;
      }
      const rec: { id?: string } = await resp.json();
      setState({ kind: 'success', id: rec.id ?? 'unknown' });
      // Reset form so a follow-up report starts clean.
      setCategory('');
      setPageUrl('');
      setDescription('');
      setEvidenceUrl('');
      setEmail('');
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Network error. Please try again.';
      setState({ kind: 'error', message: msg });
    }
  }

  const selectedCategory = CATEGORIES.find((c) => c.value === category);

  return (
    <div className="mx-auto max-w-2xl px-4 py-10 sm:px-6">
      <h1 className="font-serif text-3xl font-semibold text-civic-forest">
        Report an Error
      </h1>
      <p className="mt-2 text-sm text-civic-stone">
        Help us maintain accuracy. Every report goes through a review process.
        We never silently rewrite history — corrections create new auditable
        versions.{' '}
        <Link href="/trust" className="font-semibold text-civic-leaf hover:underline">
          See how the trust layer works →
        </Link>
      </p>

      {state.kind === 'success' ? (
        <div
          role="status"
          className="mt-8 rounded-lg border border-civic-leaf/40 bg-civic-leaf/5 p-6"
        >
          <div className="flex items-start gap-3">
            <CheckCircle2 className="mt-0.5 h-5 w-5 flex-shrink-0 text-civic-leaf" aria-hidden="true" />
            <div>
              <h2 className="font-serif text-lg font-semibold text-civic-forest">
                Thank you — your report was received.
              </h2>
              <p className="mt-1 text-sm text-civic-stone">
                Reference ID: <code className="rounded bg-civic-mist px-1.5 py-0.5 text-xs">{state.id}</code>
              </p>
              <p className="mt-2 text-xs text-civic-stone">
                A reviewer will examine the evidence you provided and decide
                whether to accept, reject, or request more information. If
                accepted, a new immutable version of the affected entity is
                created — the previous state is preserved in the audit log.
              </p>
              <div className="mt-4 flex flex-wrap gap-2">
                <button
                  type="button"
                  onClick={() => setState({ kind: 'idle' })}
                  className="inline-flex items-center gap-1.5 rounded-md border border-civic-border bg-civic-paper px-3 py-1.5 text-xs font-medium text-civic-ink hover:bg-civic-mist"
                >
                  Report another error
                </button>
                <button
                  type="button"
                  onClick={() => router.push('/trust')}
                  className="inline-flex items-center gap-1.5 rounded-md bg-civic-leaf px-3 py-1.5 text-xs font-semibold text-white hover:bg-civic-leaf/90"
                >
                  Back to Trust &amp; Evidence
                </button>
              </div>
            </div>
          </div>
        </div>
      ) : (
        <form onSubmit={onSubmit} className="mt-8 space-y-5" noValidate>
          {/* Category dropdown */}
          <div>
            <label htmlFor="category" className="block text-xs font-medium text-civic-stone">
              Category <span className="text-civic-clay">*</span>
            </label>
            <select
              id="category"
              name="category"
              value={category}
              onChange={(e) => setCategory(e.target.value)}
              aria-invalid={!!errors.category}
              aria-describedby={selectedCategory ? 'category-hint' : undefined}
              className={cn(
                'mt-1 w-full rounded-md border bg-civic-paper px-3 py-2 text-sm',
                errors.category ? 'border-civic-clay' : 'border-civic-border',
              )}
            >
              <option value="">Choose a category…</option>
              {CATEGORIES.map((c) => (
                <option key={c.value} value={c.value}>
                  {c.label}
                </option>
              ))}
            </select>
            {selectedCategory && (
              <p id="category-hint" className="mt-1.5 text-xs text-civic-stone">
                {selectedCategory.hint}
              </p>
            )}
            {errors.category && (
              <p className="mt-1 text-xs text-civic-clay">{errors.category}</p>
            )}
          </div>

          {/* Page URL */}
          <div>
            <label htmlFor="pageUrl" className="block text-xs font-medium text-civic-stone">
              Page URL with the error <span className="text-civic-clay">*</span>
            </label>
            <input
              type="url"
              id="pageUrl"
              name="pageUrl"
              value={pageUrl}
              onChange={(e) => setPageUrl(e.target.value)}
              placeholder="https://civic-intelligence.vercel.app/bills/..."
              aria-invalid={!!errors.pageUrl}
              className={cn(
                'mt-1 w-full rounded-md border bg-civic-paper px-3 py-2 text-sm',
                errors.pageUrl ? 'border-civic-clay' : 'border-civic-border',
              )}
            />
            {errors.pageUrl && (
              <p className="mt-1 text-xs text-civic-clay">{errors.pageUrl}</p>
            )}
          </div>

          {/* Description */}
          <div>
            <label htmlFor="description" className="block text-xs font-medium text-civic-stone">
              What&apos;s wrong? <span className="text-civic-clay">*</span>
            </label>
            <textarea
              id="description"
              name="description"
              rows={5}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Describe what is incorrect and what the correct information should be. Be as specific as possible (e.g. 'The publication date is shown as 2024-03-07 but should be 2024-03-15 per the Order Paper')."
              aria-invalid={!!errors.description}
              className={cn(
                'mt-1 w-full rounded-md border bg-civic-paper px-3 py-2 text-sm',
                errors.description ? 'border-civic-clay' : 'border-civic-border',
              )}
            />
            <div className="mt-1 flex items-center justify-between text-xs">
              <span className={cn(errors.description ? 'text-civic-clay' : 'text-civic-stone')}>
                {errors.description ?? 'At least 20 characters.'}
              </span>
              <span className="text-civic-stone">{description.length} chars</span>
            </div>
          </div>

          {/* Evidence URL */}
          <div>
            <label htmlFor="evidence" className="block text-xs font-medium text-civic-stone">
              Source URL with the correct information (optional)
            </label>
            <input
              type="url"
              id="evidence"
              name="evidence"
              value={evidenceUrl}
              onChange={(e) => setEvidenceUrl(e.target.value)}
              placeholder="https://www.parliament.go.ke/..."
              aria-invalid={!!errors.evidence}
              className={cn(
                'mt-1 w-full rounded-md border bg-civic-paper px-3 py-2 text-sm',
                errors.evidence ? 'border-civic-clay' : 'border-civic-border',
              )}
            />
            <p className="mt-1 flex items-center gap-1 text-xs text-civic-stone">
              <ExternalLink className="h-3 w-3" aria-hidden="true" />
              Prefer official government sources (parliament.go.ke, kenyalaw.org).
            </p>
            {errors.evidence && (
              <p className="mt-1 text-xs text-civic-clay">{errors.evidence}</p>
            )}
          </div>

          {/* Email */}
          <div>
            <label htmlFor="email" className="block text-xs font-medium text-civic-stone">
              Your email <span className="text-civic-clay">*</span>
            </label>
            <input
              type="email"
              id="email"
              name="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="you@example.com"
              aria-invalid={!!errors.email}
              className={cn(
                'mt-1 w-full rounded-md border bg-civic-paper px-3 py-2 text-sm',
                errors.email ? 'border-civic-clay' : 'border-civic-border',
              )}
            />
            <p className="mt-1 text-xs text-civic-stone">
              We&apos;ll only use this to follow up about your report.
            </p>
            {errors.email && (
              <p className="mt-1 text-xs text-civic-clay">{errors.email}</p>
            )}
          </div>

          {/* Error banner */}
          {state.kind === 'error' && (
            <div
              role="alert"
              className="flex items-start gap-2 rounded-md border border-civic-clay/40 bg-civic-clay/5 p-3 text-sm text-civic-clay"
            >
              <AlertCircle className="mt-0.5 h-4 w-4 flex-shrink-0" aria-hidden="true" />
              <div>
                <p className="font-semibold">Could not submit your report</p>
                <p className="mt-0.5 text-xs">{state.message}</p>
              </div>
            </div>
          )}

          {/* Submit */}
          <button
            type="submit"
            disabled={state.kind === 'submitting'}
            className="inline-flex w-full items-center justify-center gap-2 rounded-md bg-civic-leaf px-4 py-2.5 text-sm font-semibold text-white hover:bg-civic-leaf/90 disabled:opacity-50"
          >
            {state.kind === 'submitting' ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
                Submitting…
              </>
            ) : (
              <>
                <Send className="h-4 w-4" aria-hidden="true" />
                Submit Report
              </>
            )}
          </button>

          <p className="text-center text-[11px] text-civic-stone">
            Your report becomes a row in <code className="rounded bg-civic-mist px-1 py-0.5">trust.corrections</code> with
            status <code className="rounded bg-civic-mist px-1 py-0.5">pending</code>. Reviewers never edit the
            canonical record directly — accepted corrections create a new immutable version.
          </p>
        </form>
      )}
    </div>
  );
}
