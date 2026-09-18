'use client';

import { useState } from 'react';
import Link from 'next/link';
import { Send, Loader2, AlertTriangle, ExternalLink, LogIn, Sparkles } from 'lucide-react';
import { RealityBadge } from '@/components/reality-labels';

/**
 * AskForm — client component for the /ask page.
 *
 * POSTs the question to the Go BFF at /api/v1/questions (registered in
 * services/api/cmd/main.go via the questionsHandler wrapper that enforces
 * RequireToken + RequireScope(ScopeAIAsk)). The current handler is a
 * Q&A proxy stub — the real AI service wiring is pending (issue #19) — but
 * the shape is stable so this UI can ship today.
 *
 * Behaviour:
 *   - 401 → prompts the user to sign in via /auth/signin
 *   - 5xx / network error → friendly error with a retry hint
 *   - 200 → renders the answer with a RealityBadge kind="ASSUMPTION"
 *     (AI-GENERATED), an explicit disclaimer, and any evidence citations
 *     returned in the payload.
 *
 * The answer field is rendered verbatim — the BFF is responsible for any
 * validation_failures / citations list. We display both arrays when they
 * are present.
 */

interface QuestionCitation {
  document_id?: string;
  source_url?: string;
  page_number?: number | null;
  section?: string | null;
  snippet?: string;
  source_type?: string;
}

interface QuestionResponse {
  answer?: string;
  principal?: string;
  validated?: boolean;
  citations?: QuestionCitation[];
  validation_failures?: string[];
  // BFF Q&A proxy fields
  model?: string;
  provider?: string;
  latency_ms?: number;
}

interface AskFormProps {
  examples: string[];
}

const DISCLAIMER =
  'This answer is AI-generated. Verify against primary sources before acting on it.';

export function AskForm({ examples }: AskFormProps) {
  const [question, setQuestion] = useState('');
  const [loading, setLoading] = useState(false);
  const [response, setResponse] = useState<QuestionResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [needsAuth, setNeedsAuth] = useState(false);

  async function submit(q?: string) {
    const text = (q ?? question).trim();
    if (!text) return;
    setLoading(true);
    setError(null);
    setResponse(null);
    setNeedsAuth(false);
    if (q) setQuestion(q);

    try {
      const resp = await fetch('/api/v1/questions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
        body: JSON.stringify({ text, country: 'KE' }),
      });

      if (resp.status === 401 || resp.status === 403) {
        setNeedsAuth(true);
        setError(
          'You must be signed in to ask a question. The AI Q&A endpoint requires the `ai:ask` scope.',
        );
        return;
      }
      if (!resp.ok) {
        let detail = '';
        try {
          const j = await resp.json();
          detail = typeof j?.message === 'string' ? j.message : JSON.stringify(j);
        } catch {
          /* ignore */
        }
        setError(
          `The AI service is unavailable right now (HTTP ${resp.status}).${
            detail ? ` Detail: ${detail}` : ''
          } Please try again in a few minutes.`,
        );
        return;
      }

      const data = (await resp.json()) as QuestionResponse;
      setResponse(data);
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      setError(
        `Could not reach the AI service. The network or API may be down. (${msg})`,
      );
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="mt-8">
      {/* Question input */}
      <div className="rounded-xl border border-civic-border bg-civic-paper p-4 shadow-sm">
        <label htmlFor="ask-input" className="sr-only">
          Your question
        </label>
        <textarea
          id="ask-input"
          value={question}
          onChange={(e) => setQuestion(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && (e.metaKey || e.ctrlKey) && !loading) {
              e.preventDefault();
              submit();
            }
          }}
          placeholder="Ask any question about Kenyan civic activity…"
          rows={3}
          disabled={loading}
          className="w-full resize-none rounded-md border border-civic-border bg-civic-mist px-4 py-3 text-sm focus:outline-none focus:border-civic-leaf disabled:opacity-60"
          aria-describedby="ask-hint"
        />
        <p id="ask-hint" className="mt-2 text-xs text-civic-stone">
          Press <kbd className="rounded bg-civic-mist px-1">⌘</kbd>+
          <kbd className="rounded bg-civic-mist px-1">Enter</kbd> to submit.
        </p>
        <div className="mt-3 flex items-center justify-between gap-2">
          <p className="text-xs text-civic-stone">
            Answers are evidence-grounded. The model never fabricates citations.
          </p>
          <button
            type="button"
            onClick={() => submit()}
            disabled={loading || !question.trim()}
            className="inline-flex items-center gap-1.5 rounded-md bg-civic-leaf px-5 py-2.5 text-sm font-semibold text-white transition hover:bg-civic-leaf/90 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {loading ? (
              <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
            ) : (
              <Send className="h-4 w-4" aria-hidden="true" />
            )}
            Ask
          </button>
        </div>
      </div>

      {/* Example questions */}
      <div className="mt-6">
        <h2 className="text-xs font-semibold uppercase tracking-wide text-civic-stone">
          Try a question
        </h2>
        <ul className="mt-2 flex flex-wrap gap-2">
          {examples.map((q) => (
            <li key={q}>
              <button
                type="button"
                onClick={() => submit(q)}
                disabled={loading}
                className="rounded-full bg-civic-mist px-3 py-1 text-sm text-civic-stone transition hover:bg-civic-leaf/10 hover:text-civic-leaf disabled:cursor-not-allowed disabled:opacity-50"
              >
                {q}
              </button>
            </li>
          ))}
        </ul>
      </div>

      {/* Error / auth required */}
      {error && (
        <div
          className="mt-6 rounded-lg border border-amber-300 bg-amber-50 p-4 text-sm text-amber-900"
          role="alert"
        >
          <div className="flex items-start gap-2">
            <AlertTriangle className="mt-0.5 h-4 w-4 flex-shrink-0" aria-hidden="true" />
            <div className="flex-1">
              <p>{error}</p>
              {needsAuth && (
                <Link
                  href="/auth/signin"
                  className="mt-3 inline-flex items-center gap-1.5 rounded-md bg-civic-forest px-3 py-1.5 text-xs font-semibold text-civic-paper hover:bg-civic-leaf"
                >
                  <LogIn className="h-3.5 w-3.5" aria-hidden="true" />
                  Sign in to ask
                </Link>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Answer */}
      {response && (
        <div
          className="mt-8 rounded-xl border border-civic-border bg-civic-paper p-5 shadow-sm"
          aria-live="polite"
        >
          <div className="flex flex-wrap items-center gap-2">
            <RealityBadge kind="ASSUMPTION" size="md">
              <span className="ml-1 normal-case tracking-normal">AI-GENERATED</span>
            </RealityBadge>
            <span className="inline-flex items-center gap-1 text-xs text-civic-stone">
              <Sparkles className="h-3 w-3" aria-hidden="true" />
              {response.provider ?? 'civic-ai'} · {response.model ?? 'qanda-proxy'}
            </span>
            {typeof response.latency_ms === 'number' && (
              <span className="text-xs text-civic-stone">
                {response.latency_ms}ms
              </span>
            )}
            {typeof response.validated === 'boolean' && (
              <span className="text-xs text-civic-stone">
                validated: {response.validated ? 'yes' : 'no'}
              </span>
            )}
          </div>

          <p className="mt-3 whitespace-pre-wrap text-civic-ink">
            {response.answer ?? 'No answer was returned.'}
          </p>

          {/* Disclaimer */}
          <p className="mt-4 rounded-md bg-civic-acacia/10 px-3 py-2 text-xs text-civic-clay">
            {DISCLAIMER}
          </p>

          {/* Citations */}
          {response.citations && response.citations.length > 0 && (
            <details className="mt-4 rounded-md bg-civic-mist p-3 text-sm">
              <summary className="cursor-pointer font-medium text-civic-ink">
                Evidence citations ({response.citations.length})
              </summary>
              <ul className="mt-2 space-y-2 text-xs">
                {response.citations.map((c, i) => (
                  <li key={i} className="border-l-2 border-civic-leaf/40 pl-3">
                    {c.source_url ? (
                      <a
                        href={c.source_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex items-center gap-1 font-medium text-civic-leaf hover:underline"
                      >
                        [{i + 1}] {c.source_type ?? 'source'}
                        {c.page_number ? ` p.${c.page_number}` : ''}
                        {c.section ? ` §${c.section}` : ''}
                        <ExternalLink className="h-3 w-3" aria-hidden="true" />
                      </a>
                    ) : (
                      <span className="font-medium text-civic-ink">
                        [{i + 1}] {c.source_type ?? 'source'}
                      </span>
                    )}
                    {c.snippet && (
                      <span className="mt-1 block italic text-civic-stone">
                        &ldquo;{c.snippet}&rdquo;
                      </span>
                    )}
                  </li>
                ))}
              </ul>
            </details>
          )}

          {/* Validation failures */}
          {response.validation_failures && response.validation_failures.length > 0 && (
            <details className="mt-3 rounded-md bg-rose-50 p-3 text-xs text-rose-800">
              <summary className="cursor-pointer font-medium">
                Validation failures ({response.validation_failures.length})
              </summary>
              <ul className="mt-2 list-disc pl-4">
                {response.validation_failures.map((f, i) => (
                  <li key={i}>{f}</li>
                ))}
              </ul>
            </details>
          )}
        </div>
      )}
    </div>
  );
}
