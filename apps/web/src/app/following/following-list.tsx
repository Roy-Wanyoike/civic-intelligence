'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { Bell, Loader2, Trash2, FileText, Users, Building2, Hash, User } from 'lucide-react';
import { deleteSubscription, listSubscriptions, ApiError } from '@/lib/api';
import type { Subscription } from '@/lib/api';

const ENTITY_ICONS: Record<string, typeof FileText> = {
  bill: FileText,
  committee: Users,
  topic: Hash,
  institution: Building2,
  person: User,
};

function entityHref(s: Subscription): string {
  switch (s.entity_type) {
    case 'bill': return `/bills/${s.entity_id}`;
    case 'committee': return `/committees#${s.entity_id}`;
    case 'topic': return `/topics#${s.entity_id}`;
    case 'institution': return `/institutions#${s.entity_id}`;
    case 'person': return `/people#${s.entity_id}`;
    default: return '#';
  }
}

/**
 * FollowingList renders the caller's subscriptions, grouped by entity_type.
 *
 * Until the identity service (#20) issues real OIDC tokens, this component
 * gracefully renders an empty state — it does not block the page from
 * rendering. When a token is available (e.g., from localStorage), it is sent
 * as the Bearer token to the Go BFF.
 */
export function FollowingList() {
  const [items, setItems] = useState<Subscription[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [token, setToken] = useState<string | undefined>(undefined);

  useEffect(() => {
    // Try to read a token from localStorage — set by the (forthcoming) sign-in flow.
    try {
      const stored = typeof window !== 'undefined' ? window.localStorage.getItem('civic.token') : null;
      if (stored) setToken(stored);
    } catch {
      /* localStorage not available */
    }
  }, []);

  useEffect(() => {
    if (token === undefined) return; // still loading
    if (!token) {
      setLoading(false);
      return;
    }
    let cancelled = false;
    (async () => {
      setLoading(true);
      try {
        const resp = await listSubscriptions({ token });
        if (cancelled) return;
        setItems(resp.items);
      } catch (err) {
        if (cancelled) return;
        setError(err instanceof ApiError ? `HTTP ${err.status}` : 'Failed to load follows');
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [token]);

  async function handleUnfollow(id: string) {
    if (!token) return;
    try {
      await deleteSubscription({ id, token });
      setItems((prev) => prev.filter((s) => s.id !== id));
    } catch (err) {
      setError(err instanceof ApiError ? `HTTP ${err.status}` : 'Unfollow failed');
    }
  }

  if (loading) {
    return (
      <div className="flex items-center gap-2 rounded-lg border border-civic-border p-6 text-sm text-civic-stone">
        <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
        Loading your follows…
      </div>
    );
  }

  if (!token) {
    return (
      <div className="rounded-lg border border-dashed border-civic-border p-6 text-sm text-civic-stone">
        <p className="flex items-center gap-2">
          <Bell className="h-4 w-4" aria-hidden="true" />
          Sign in to see your follows.
        </p>
        <p className="mt-1">
          You can still browse Bills, committees, and topics publicly — the Follow button will ask you
          to sign in once the identity service is live (issue #20).
        </p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-lg border border-civic-clay/30 bg-civic-clay/5 p-6 text-sm text-civic-clay">
        {error}
      </div>
    );
  }

  if (items.length === 0) {
    return (
      <div className="rounded-lg border border-dashed border-civic-border p-12 text-center text-sm text-civic-stone">
        You are not following anything yet.
        <div className="mt-3">
          <Link href="/bills" className="inline-flex items-center gap-1 text-civic-leaf hover:underline">
            <FileText className="h-4 w-4" aria-hidden="true" />
            Browse Bills to follow →
          </Link>
        </div>
      </div>
    );
  }

  // Group by entity_type.
  const groups: Record<string, Subscription[]> = {};
  for (const s of items) {
    if (!groups[s.entity_type]) groups[s.entity_type] = [];
    groups[s.entity_type].push(s);
  }

  return (
    <div className="space-y-6">
      {Object.entries(groups).map(([type, follows]) => {
        const Icon = ENTITY_ICONS[type] ?? Bell;
        return (
          <section key={type}>
            <h2 className="mb-3 flex items-center gap-2 font-serif text-lg font-semibold text-civic-ink">
              <Icon className="h-4 w-4 text-civic-leaf" aria-hidden="true" />
              {type.charAt(0).toUpperCase() + type.slice(1)}s
              <span className="text-xs font-normal text-civic-stone">({follows.length})</span>
            </h2>
            <ul className="grid gap-3 sm:grid-cols-2">
              {follows.map((s) => (
                <li key={s.id} className="flex items-start justify-between gap-2 rounded-lg border border-civic-border bg-civic-paper p-3">
                  <Link href={entityHref(s)} className="block flex-1 hover:underline">
                    <p className="font-mono text-xs text-civic-stone">{s.entity_id}</p>
                    <p className="mt-1 text-xs text-civic-stone">
                      Followed {new Date(s.created_at).toLocaleDateString('en-KE', { year: 'numeric', month: 'short', day: 'numeric' })}
                    </p>
                  </Link>
                  <button
                    type="button"
                    onClick={() => handleUnfollow(s.id)}
                    aria-label="Unfollow"
                    className="rounded-md p-1 text-civic-stone hover:bg-civic-clay/10 hover:text-civic-clay"
                  >
                    <Trash2 className="h-4 w-4" aria-hidden="true" />
                  </button>
                </li>
              ))}
            </ul>
          </section>
        );
      })}
    </div>
  );
}
