'use client';

/**
 * LiveParliamentBanner — "🔴 Parliament is live" banner + modal (issue #283).
 *
 * Server components can't render the modal (which needs client-side state
 * to open/close), so the homepage fetches `/api/v1/calendar/live` at
 * request time and passes the result to this client component. When
 * `live.is_live` is false, the component renders nothing — the homepage
 * stays clean during parliamentary recesses.
 *
 * Clicking the banner opens a modal with the YouTube embed. The modal:
 *
 *   - traps focus (role="dialog" aria-modal="true")
 *   - closes on backdrop click, Escape key, or the close button
 *   - locks body scroll while open (so the page behind doesn't scroll)
 *   - restores focus to the banner button on close (keyboard a11y)
 *
 * Accessibility:
 *   - The banner is a real <button> (not a div) so keyboard users can
 *     tab to it and press Enter to open the modal.
 *   - The modal's aria-label carries the sitting title so screen readers
 *     announce "Parliament is live — National Assembly — Morning Sitting".
 *   - The close button has an aria-label ("Close livestream modal") so
 *     it's announced as a close action rather than a bare "X".
 */

import { useEffect, useRef, useState } from 'react';
import { Radio, X } from 'lucide-react';
import {
  YouTubeEmbed,
  extractYouTubeVideoId,
} from '@/components/youtube-embed';
import type { CalendarLive } from '@/lib/api';

interface LiveParliamentBannerProps {
  /** The live plenary status (fetched server-side on the homepage). */
  live: CalendarLive;
  /**
   * Optional className applied to the outer banner wrapper so the
   * homepage can adjust the banner's container width / margin without
   * forking the component.
   */
  className?: string;
}

/**
 * LiveParliamentBanner renders the live banner + the modal that opens on
 * click. Returns null when `live.is_live` is false so the homepage can
 * always render `<LiveParliamentBanner live={live} />` without an extra
 * null guard.
 */
export function LiveParliamentBanner({
  live,
  className,
}: LiveParliamentBannerProps) {
  const [open, setOpen] = useState(false);
  const triggerRef = useRef<HTMLButtonElement>(null);

  // Escape closes the modal (in addition to the close button + backdrop).
  useEffect(() => {
    if (!open) return;
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault();
        setOpen(false);
      }
    };
    document.addEventListener('keydown', handleKey);
    return () => document.removeEventListener('keydown', handleKey);
  }, [open]);

  // Restore focus to the banner button on close (keyboard a11y).
  useEffect(() => {
    if (!open && triggerRef.current) {
      const id = window.setTimeout(() => {
        triggerRef.current?.focus();
      }, 0);
      return () => window.clearTimeout(id);
    }
    return undefined;
  }, [open]);

  // Lock body scroll while the modal is open so the page behind doesn't
  // scroll. Restored on close (or unmount).
  useEffect(() => {
    if (!open) return;
    const prev = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      document.body.style.overflow = prev;
    };
  }, [open]);

  if (!live.is_live) return null;

  const videoId = extractYouTubeVideoId(live.video_url);
  const title =
    live.title ?? `${live.house ?? 'Parliament'} Sitting — ${live.date ?? 'today'}`;
  const house = live.house ?? 'Parliament';

  return (
    <>
      {/* Banner — the only always-visible piece. Clicking it opens the modal. */}
      <button
        ref={triggerRef}
        type="button"
        onClick={() => setOpen(true)}
        aria-haspopup="dialog"
        aria-expanded={open}
        aria-label={`Parliament is live now — ${title}. Click to watch the stream.`}
        className={
          'group flex w-full items-center gap-3 rounded-lg border border-red-300 bg-red-50 px-4 py-3 text-left transition hover:border-red-400 hover:bg-red-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-red-500 focus-visible:ring-offset-2 ' +
          (className ?? '')
        }
      >
        <span
          className="relative flex h-3 w-3 flex-shrink-0"
          aria-hidden="true"
        >
          {/* Pulsing red dot to signal "live". */}
          <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-red-500 opacity-75" />
          <span className="relative inline-flex h-3 w-3 rounded-full bg-red-600" />
        </span>
        <Radio className="h-5 w-5 flex-shrink-0 text-red-600" aria-hidden="true" />
        <span className="flex flex-1 flex-wrap items-baseline gap-x-2 gap-y-0.5 text-sm">
          <span className="font-semibold text-red-800">
            🔴 Parliament is live — Watch now
          </span>
          <span className="text-red-700/80">
            {house}
            {live.date ? ` · ${live.date}` : ''}
          </span>
        </span>
        <span className="text-xs font-medium text-red-700 group-hover:underline">
          Watch stream →
        </span>
      </button>

      {/* Modal — rendered only when open. Uses the same backdrop pattern
          as the Command Palette (apps/web/src/components/command-palette.tsx). */}
      {open && (
        <div
          role="dialog"
          aria-modal="true"
          aria-label={`Parliament is live — ${title}`}
          className="fixed inset-0 z-[100] flex items-start justify-center bg-civic-ink/60 backdrop-blur-sm sm:items-center"
          // Click on the backdrop closes the modal. Clicks inside the
          // dialog panel do NOT bubble out (panel stops propagation).
          onClick={(e) => {
            if (e.target === e.currentTarget) setOpen(false);
          }}
        >
          <div className="mt-[10vh] w-[min(48rem,calc(100vw-2rem))] overflow-hidden rounded-xl border border-civic-border bg-civic-paper shadow-2xl sm:mt-0">
            {/* Modal header — sitting title + close button. */}
            <div className="flex items-start justify-between gap-4 border-b border-civic-border px-5 py-4">
              <div className="min-w-0">
                <div className="flex items-center gap-2 text-xs text-red-700">
                  <span className="relative flex h-2 w-2" aria-hidden="true">
                    <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-red-500 opacity-75" />
                    <span className="relative inline-flex h-2 w-2 rounded-full bg-red-600" />
                  </span>
                  <span className="font-semibold uppercase tracking-wide">
                    Live now
                  </span>
                  {house && (
                    <span className="text-civic-stone">· {house}</span>
                  )}
                </div>
                <h2 className="mt-1 font-serif text-lg font-semibold text-civic-forest">
                  {title}
                </h2>
                {live.date && (
                  <p className="mt-0.5 text-xs text-civic-stone">
                    Sitting of {live.date}
                  </p>
                )}
              </div>
              <button
                type="button"
                onClick={() => setOpen(false)}
                aria-label="Close livestream modal"
                className="flex-shrink-0 rounded-md p-1.5 text-civic-stone hover:bg-civic-mist hover:text-civic-ink focus:outline-none focus-visible:ring-2 focus-visible:ring-civic-leaf"
              >
                <X className="h-5 w-5" aria-hidden="true" />
              </button>
            </div>

            {/* Modal body — the YouTube embed. Autoplay is on so the
                stream starts the moment the user opts in (the click on
                the banner is the user gesture modern browsers require
                before autoplay with sound is permitted). */}
            <div className="p-5">
              {videoId ? (
                <YouTubeEmbed
                  videoId={videoId}
                  title={title}
                  autoplay
                  className="w-full"
                />
              ) : (
                <p className="rounded-md border border-dashed border-civic-border p-4 text-sm text-civic-stone">
                  The live stream URL could not be parsed. The sitting is
                  in progress; please try the parliament&apos;s YouTube
                  channel directly.
                </p>
              )}
              <p className="mt-3 text-xs text-civic-stone">
                Streamed live from the {house}. The video is hosted on
                YouTube; closing this modal pauses playback.
              </p>
            </div>
          </div>
        </div>
      )}
    </>
  );
}

export default LiveParliamentBanner;
