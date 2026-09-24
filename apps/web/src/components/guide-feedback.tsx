'use client';

import { useState } from 'react';
import { ThumbsUp, ThumbsDown, CheckCircle2 } from 'lucide-react';

/**
 * GuideFeedback — "Was this helpful?" widget rendered at the bottom of
 * every guide.
 *
 * This is a deliberately lightweight client component: a single state
 * machine (idle → submitted) and two buttons. It does NOT make a network
 * call — feedback is captured client-side only at this stage. Wiring to
 * the platform's notification/subscription backend is a follow-up task;
 * the data shape (`{slug, helpful: boolean, comment?}`) is what a future
 * POST endpoint would receive.
 *
 * Why client-side only for now: the existing platform has no feedback
 * endpoint, and shipping a half-wired fetch that 404s would be worse than
 * capturing the state locally. Once a `/api/v1/learn/feedback` route
 * exists, swap the `setSubmitted(true)` call for a `fetch` POST.
 */
export function GuideFeedback({ slug }: { slug: string }) {
  const [submitted, setSubmitted] = useState(false);
  const [helpful, setHelpful] = useState<boolean | null>(null);
  const [comment, setComment] = useState('');

  function submit(value: boolean) {
    setHelpful(value);
    // Mark as submitted only after the user has picked a side; the
    // optional comment textarea appears after the choice, and the user
    // can submit-without-comment by clicking "Submit feedback" or just
    // skip it (the choice itself is enough signal).
    // (No network call — see component docstring.)
    setSubmitted(true);
  }

  if (submitted) {
    return (
      <div
        className="rounded-xl border border-civic-border bg-civic-mist/50 p-6"
        role="status"
        aria-live="polite"
      >
        <div className="flex items-start gap-3">
          <CheckCircle2
            className="mt-0.5 h-5 w-5 flex-shrink-0 text-civic-leaf"
            aria-hidden="true"
          />
          <div className="min-w-0 flex-1">
            <h3 className="font-serif text-base font-semibold text-civic-forest">
              Thanks for the feedback
            </h3>
            <p className="mt-1 text-sm text-civic-stone">
              {helpful
                ? 'We\'re glad this guide was useful. Your input helps us prioritise new guides and updates.'
                : 'Thanks for telling us. We use this to identify guides that need clearer wording, more examples, or updated sources.'}
            </p>
            {comment.trim() && (
              <p className="mt-2 rounded-md bg-civic-paper px-3 py-2 text-sm text-civic-ink">
                <span className="text-xs font-semibold text-civic-stone">
                  Your note:
                </span>{' '}
                {comment.trim()}
              </p>
            )}
            <button
              type="button"
              onClick={() => {
                setSubmitted(false);
                setHelpful(null);
                setComment('');
              }}
              className="mt-3 text-xs text-civic-leaf hover:underline"
            >
              Submit another response
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="rounded-xl border border-civic-border bg-civic-mist/50 p-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h3 className="font-serif text-base font-semibold text-civic-forest">
            Was this helpful?
          </h3>
          <p className="mt-1 text-sm text-civic-stone">
            Your feedback shapes which guides we update next.
          </p>
        </div>
        <div className="flex flex-shrink-0 gap-2" data-guide-slug={slug}>
          <button
            type="button"
            onClick={() => submit(true)}
            aria-pressed={helpful === true}
            className="inline-flex items-center gap-1.5 rounded-full border border-civic-border bg-civic-paper px-4 py-2 text-sm font-medium text-civic-ink transition hover:border-civic-leaf hover:text-civic-leaf"
          >
            <ThumbsUp className="h-4 w-4" aria-hidden="true" />
            Yes
          </button>
          <button
            type="button"
            onClick={() => submit(false)}
            aria-pressed={helpful === false}
            className="inline-flex items-center gap-1.5 rounded-full border border-civic-border bg-civic-paper px-4 py-2 text-sm font-medium text-civic-ink transition hover:border-civic-clay hover:text-civic-clay"
          >
            <ThumbsDown className="h-4 w-4" aria-hidden="true" />
            No
          </button>
        </div>
      </div>
      <form
        className="mt-4"
        onSubmit={(e) => {
          e.preventDefault();
          // Submitting the optional comment with no helpful choice is a
          // no-op — the choice buttons are the primary signal.
          if (helpful === null) return;
          setSubmitted(true);
        }}
      >
        <label htmlFor="guide-feedback-comment" className="sr-only">
          Optional: tell us what could be better
        </label>
        <textarea
          id="guide-feedback-comment"
          name="comment"
          rows={2}
          value={comment}
          onChange={(e) => setComment(e.target.value)}
          placeholder="Optional: tell us what could be clearer, missing, or out of date."
          className="w-full rounded-md border border-civic-border bg-civic-paper px-3 py-2 text-sm text-civic-ink placeholder:text-civic-stone focus:border-civic-leaf focus:outline-none focus:ring-1 focus:ring-civic-leaf"
        />
        {helpful !== null && (
          <button
            type="submit"
            className="mt-2 inline-flex items-center gap-1.5 rounded-md bg-civic-leaf px-4 py-2 text-sm font-semibold text-white transition hover:bg-civic-leaf/90"
          >
            Submit feedback
          </button>
        )}
      </form>
    </div>
  );
}
