'use client';

import { useState } from 'react';

interface FollowLawButtonProps {
  actId: string;
}

export function FollowLawButton({ actId }: FollowLawButtonProps) {
  const [followed, setFollowed] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleFollow = async () => {
    setLoading(true);
    setError(null);
    try {
      const resp = await fetch(`/api/v1/acts/${actId}/follow`, { method: 'POST' });
      if (!resp.ok) {
        throw new Error(`HTTP ${resp.status}`);
      }
      setFollowed(true);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to follow');
    } finally {
      setLoading(false);
    }
  };

  if (followed) {
    return (
      <div className="mt-3 rounded-lg border border-emerald-300 bg-emerald-100 p-3 text-sm text-emerald-900">
        You are now following this law. The platform will notify you when
        authoritative evidence indicates something changed.
      </div>
    );
  }

  return (
    <div className="mt-3">
      <button
        type="button"
        onClick={handleFollow}
        disabled={loading}
        className="inline-flex items-center gap-2 rounded-lg bg-violet-700 px-4 py-2 text-sm font-semibold text-white transition hover:bg-violet-800 disabled:opacity-50"
      >
        {loading ? 'Following...' : 'Follow this law'}
      </button>
      {error && (
        <p className="mt-2 text-xs text-rose-700">{error}</p>
      )}
    </div>
  );
}
