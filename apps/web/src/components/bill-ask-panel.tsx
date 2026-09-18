'use client';

import { useState } from 'react';
import { Send, Loader2, AlertTriangle, CheckCircle2 } from 'lucide-react';
import { askQuestion, ApiError } from '@/lib/api';
import type { AIResponse } from '@/lib/types';

const SAMPLE_QUESTIONS = [
  'What is this Bill about in plain language?',
  'What stage is it at and what happens next?',
  'Who is examining it?',
  'What changed in the latest version?',
  'How might this affect small businesses?',
];

export function BillAskPanel({
  billId,
  expanded = false,
}: {
  billId: string;
  expanded?: boolean;
}) {
  const [question, setQuestion] = useState('');
  const [loading, setLoading] = useState(false);
  const [response, setResponse] = useState<AIResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  async function submit(q?: string) {
    const text = (q ?? question).trim();
    if (!text) return;
    setLoading(true);
    setError(null);
    setResponse(null);
    try {
      const r = await askQuestion(text, { billId });
      setResponse(r);
      if (q) setQuestion(q);
    } catch (e) {
      if (e instanceof ApiError) {
        setError(`The AI service is unavailable (${e.status}). Please try again later.`);
      } else {
        setError('An unexpected error occurred. Please try again.');
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="mt-4 rounded-lg border border-civic-border bg-civic-paper p-4">
      <div className="flex gap-2">
        <input
          type="text"
          value={question}
          onChange={(e) => setQuestion(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !loading) submit();
          }}
          placeholder="Ask a question about this Bill…"
          aria-label="Ask a question about this Bill"
          className="flex-1 rounded-md border border-civic-border bg-civic-mist px-3 py-2 text-sm focus:outline-none focus:border-civic-leaf"
          disabled={loading}
        />
        <button
          type="button"
          onClick={() => submit()}
          disabled={loading || !question.trim()}
          className="inline-flex items-center gap-1.5 rounded-md bg-civic-leaf px-4 py-2 text-sm font-semibold text-white hover:bg-civic-leaf/90 disabled:opacity-50"
        >
          {loading ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" /> : <Send className="h-4 w-4" aria-hidden="true" />}
          Ask
        </button>
      </div>

      {expanded && (
        <div className="mt-3 flex flex-wrap gap-1.5">
          {SAMPLE_QUESTIONS.map((q) => (
            <button
              key={q}
              type="button"
              onClick={() => submit(q)}
              disabled={loading}
              className="rounded-full bg-civic-mist px-3 py-1 text-xs text-civic-stone hover:bg-civic-leaf/10 hover:text-civic-leaf"
            >
              {q}
            </button>
          ))}
        </div>
      )}

      {error && (
        <div className="mt-4 flex items-start gap-2 rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">
          <AlertTriangle className="mt-0.5 h-4 w-4 flex-shrink-0" aria-hidden="true" />
          <p>{error}</p>
        </div>
      )}

      {response && (
        <div className="mt-4 border-t border-civic-border pt-4">
          <div className="flex items-center gap-2 text-sm">
            {response.validated ? (
              <>
                <CheckCircle2 className="h-4 w-4 text-civic-leaf" aria-hidden="true" />
                <span className="font-medium text-civic-leaf">Verified answer</span>
              </>
            ) : (
              <>
                <AlertTriangle className="h-4 w-4 text-civic-acacia" aria-hidden="true" />
                <span className="font-medium text-civic-acacia">
                  Response requires review — some claims could not be fully verified
                </span>
              </>
            )}
            <span className="ml-auto text-xs text-civic-stone">
              {response.provider} · {response.latency_ms}ms · {response.citations.length} citation(s)
            </span>
          </div>
          <p className="mt-3 whitespace-pre-wrap text-civic-ink">{response.answer}</p>

          {response.validation_failures.length > 0 && (
            <details className="mt-3 rounded-md bg-civic-mist p-2 text-xs text-civic-stone">
              <summary className="cursor-pointer">Validation failures ({response.validation_failures.length})</summary>
              <ul className="mt-2 list-disc pl-4">
                {response.validation_failures.map((f, i) => (<li key={i}>{f}</li>))}
              </ul>
            </details>
          )}

          {response.citations.length > 0 && (
            <details className="mt-3 rounded-md bg-civic-mist p-2 text-xs text-civic-stone">
              <summary className="cursor-pointer">Citations ({response.citations.length})</summary>
              <ul className="mt-2 space-y-1">
                {response.citations.map((c, i) => (
                  <li key={i}>
                    <a href={c.source_url} target="_blank" rel="noopener noreferrer" className="text-civic-leaf hover:underline">
                      [{i + 1}] {c.source_type}
                      {c.page_number ? ` p.${c.page_number}` : ''}
                      {c.section ? ` §${c.section}` : ''}
                    </a>
                    <span className="block italic text-civic-stone">&ldquo;{c.snippet}&rdquo;</span>
                  </li>
                ))}
              </ul>
            </details>
          )}
        </div>
      )}
    </div>
  );
}
