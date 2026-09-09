'use client';

import { useEffect, useState } from 'react';
import { Bell, BellOff, Loader2 } from 'lucide-react';
import { createSubscription, deleteSubscription, listSubscriptions } from '@/lib/api';
import type { Subscription } from '@/lib/api';

interface FollowButtonProps {
  /** Entity type: bill | committee | topic | institution | person */
  entityType: 'bill' | 'committee' | 'topic' | 'institution' | 'person';
  /** Entity ID — usually a UUID or slug. */
  entityId: string;
  /** Optional Bearer token (when the caller is signed in). */
  token?: string;
  /** Optional label override (defaults to "Follow"). */
  label?: string;
  /** Optional callback fired after a successful follow / unfollow. */
  onFollowChange?: (isFollowing: boolean) => void;
}

/**
 * FollowButton toggles a user's subscription to an entity (issue #110).
 *
 * The Go API endpoint POST /api/v1/subscriptions is idempotent, so re-clicking
 * "Follow" when already following is a no-op (returns the existing record).
 * Unfollow uses DELETE /api/v1/subscriptions/{id} which is scoped to the
 * caller — only the owner can delete their own follows.
 */
export function FollowButton({
  entityType,
  entityId,
  token,
  label = 'Follow this Bill',
  onFollowChange,
}: FollowButtonProps) {
  const [following, setFollowing] = useState(false);
  const [subscriptionId, setSubscriptionId] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // On mount, check whether the caller already follows this entity.
  useEffect(() => {
    let cancelled = false;
    if (!token || !entityId) return;
    (async () => {
      try {
        const resp = await listSubscriptions({ entityType, token });
        if (cancelled) return;
        const match = resp.items.find((s: Subscription) => s.entity_type === entityType && s.entity_id === entityId);
        if (match) {
          setFollowing(true);
          setSubscriptionId(match.id);
        }
      } catch {
        // Silently ignore — anonymous users just see the default "Follow" state.
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [entityType, entityId, token]);

  async function toggle() {
    if (loading) return;
    setLoading(true);
    setError(null);
    try {
      if (following && subscriptionId) {
        await deleteSubscription({ id: subscriptionId, token });
        setFollowing(false);
        setSubscriptionId(null);
        onFollowChange?.(false);
      } else {
        const sub = await createSubscription({ entityType, entityId, token });
        setFollowing(true);
        setSubscriptionId(sub.id);
        onFollowChange?.(true);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Action failed');
    } finally {
      setLoading(false);
    }
  }

  const Icon = following ? BellOff : Bell;
  const text = following ? 'Following' : label;

  return (
    <div className="flex flex-col gap-1">
      <button
        type="button"
        onClick={toggle}
        disabled={loading}
        aria-pressed={following}
        className="inline-flex w-full items-center justify-center gap-1.5 rounded-md bg-civic-leaf px-3 py-2 text-sm font-semibold text-white hover:bg-civic-leaf/90 disabled:opacity-50"
      >
        {loading ? (
          <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
        ) : (
          <Icon className="h-4 w-4" aria-hidden="true" />
        )}
        {text}
      </button>
      {error && <p className="text-xs text-civic-clay">{error}</p>}
      {!token && (
        <p className="text-xs text-civic-stone">Sign in to follow this Bill and get notifications.</p>
      )}
    </div>
  );
}
