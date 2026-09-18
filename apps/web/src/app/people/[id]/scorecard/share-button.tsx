'use client';

import { Share2, Check } from 'lucide-react';
import { useState } from 'react';

interface Props {
  name: string;
  personId: string;
}

/**
 * ScorecardShareButton generates a shareable URL for the MP scorecard and
 * copies it to the clipboard. The URL points at the canonical frontend route
 * (`/people/{id}/scorecard`) so external linkers always land on the scorecard
 * page rather than the API JSON.
 *
 * On successful copy, the button label flips to "Copied!" for 2 seconds so
 * the user gets confirmation feedback without an alert dialog.
 */
export function ScorecardShareButton({ name, personId }: Props) {
  const [copied, setCopied] = useState(false);

  async function handleShare() {
    if (typeof window === 'undefined') return;
    const url = `${window.location.origin}/people/${encodeURIComponent(personId)}/scorecard`;
    try {
      if (navigator.share) {
        // Web Share API — preferred on mobile (no clipboard permission prompt).
        await navigator.share({
          title: `${name} — MP Scorecard`,
          text: `MP scorecard for ${name} — factual parliamentary record.`,
          url,
        });
        return;
      }
      await navigator.clipboard.writeText(url);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Fallback: open a mailto: link so the user can still share manually.
      window.location.href = `mailto:?subject=${encodeURIComponent(name + ' — MP Scorecard')}&body=${encodeURIComponent(url)}`;
    }
  }

  return (
    <button
      type="button"
      onClick={handleShare}
      className="inline-flex items-center gap-1 rounded-md border border-civic-border px-3 py-1.5 text-sm hover:bg-civic-mist"
      aria-label={`Share scorecard for ${name}`}
    >
      {copied ? (
        <>
          <Check className="h-4 w-4" aria-hidden="true" />
          Copied!
        </>
      ) : (
        <>
          <Share2 className="h-4 w-4" aria-hidden="true" />
          Share
        </>
      )}
    </button>
  );
}
