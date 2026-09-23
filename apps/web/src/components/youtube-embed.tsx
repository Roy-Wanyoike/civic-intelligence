'use client';

/**
 * YouTubeEmbed — responsive 16:9 YouTube iframe player (issue #283).
 *
 * Used in two contexts:
 *
 *   1. Homepage "Parliament is live" modal — the modal opens when the user
 *      clicks the live banner; the embed plays the live plenary stream.
 *   2. Bill detail page — when the Bill's `video_url` field is present (i.e.
 *      the Bill was discussed in a recent Hansard sitting), the embed renders
 *      below the Bill title so citizens can watch the actual debate.
 *
 * Accepts a `videoId` prop (extracted from the YouTube URL by the caller
 * via the exported `extractYouTubeVideoId` helper) rather than a raw URL
 * so the same component works for `youtu.be/<id>`, `youtube.com/watch?v=<id>`,
 * and `youtube.com/embed/<id>` links without re-parsing on every render.
 *
 * The iframe is wrapped in a 16:9 aspect-ratio container (`aspect-video`)
 * so the player scales responsively with its parent — the embed never
 * overflows on mobile and never wastes space on desktop. `loading="lazy"`
 * keeps the iframe off the critical render path (a Bill page can have
 * many sections and the embed is below the fold).
 *
 * Accessibility:
 *   - `title` is set on the iframe so screen readers announce "YouTube
 *     video player" + the descriptive title passed by the caller.
 *   - The "Watch on YouTube" link is a real anchor (`<a>`) so keyboard
 *     users can tab to it and right-click / long-press to copy the link.
 *   - The iframe carries `allow="autoplay; ..."` so YouTube can start
 *     playback immediately when the user clicks the live banner (autoplay
 *     is gated by the user gesture in modern browsers, but the permission
 *     policy header must permit it).
 */

import { ExternalLink } from 'lucide-react';

export interface YouTubeEmbedProps {
  /** YouTube video ID (e.g., "dQw4w9WgXcQ"). NOT the full URL. */
  videoId: string;
  /**
   * Optional descriptive title for the iframe — surfaced as the `title`
   * attribute so screen readers announce the embed's purpose. Defaults
   * to a generic "YouTube video player" when not provided.
   */
  title?: string;
  /**
   * Optional className applied to the outer wrapper so callers can
   * adjust spacing / sizing without forking the component.
   */
  className?: string;
  /**
   * Whether to autoplay the video when the iframe loads. Autoplay is
   * gated by the user gesture in modern browsers; this flag just sets
   * the `autoplay=1` query parameter on the embed URL. Default false
   * (muted autoplay would be more reliable but is intentionally NOT
   * done — a Bill page's embed should be silent until the user opts in).
   */
  autoplay?: boolean;
  /**
   * Whether to render the "Watch on YouTube" link below the embed.
   * Default true. Callers may suppress the link when the embed is
   * rendered in a context where the link would be redundant (e.g.,
   * a modal that already has a "Open on YouTube" action button).
   */
  showWatchLink?: boolean;
}

// youTubeEmbedAllow is the iframe `allow` permission policy string. The
// list matches the task spec exactly (issue #283):
//   - accelerometer          — used by YouTube's player UI for orientation
//   - autoplay               — permits autoplay when the caller opts in
//   - clipboard-write        — lets the player copy share URLs to clipboard
//   - encrypted-media       — required for DRM-protected content
//   - gyroscope             — used by YouTube's mobile player
//   - picture-in-picture     — lets the user pop the video out
const youTubeEmbedAllow =
  'accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture';

// youTubeNoCookieDomain is the privacy-enhanced YouTube embed host. We
// use youtube-nocookie.com so YouTube does not set tracking cookies on
// page load (only when the user actually plays the video). This is the
// recommended pattern for civic-tech sites that may be subject to GDPR /
// Kenya DPA 2019 consent requirements.
const youTubeNoCookieDomain = 'https://www.youtube-nocookie.com';

/**
 * extractYouTubeVideoId parses a YouTube video ID out of any of the
 * common YouTube URL forms:
 *
 *   - https://youtu.be/dQw4w9WgXcQ
 *   - https://www.youtube.com/watch?v=dQw4w9WgXcQ
 *   - https://www.youtube.com/embed/dQw4w9WgXcQ
 *   - https://www.youtube.com/shorts/dQw4w9WgXcQ
 *   - https://m.youtube.com/watch?v=dQw4w9WgXcQ
 *
 * Returns the empty string when the input cannot be parsed as a YouTube
 * URL. Callers MUST treat the empty string as "no embed to render" — the
 * component itself does NOT render anything when videoId is falsy so the
 * empty string short-circuits cleanly.
 *
 * Also accepts a bare video ID (11 characters, alphanumeric + dash + underscore)
 * as a pass-through so callers that already have the ID don't have to
 * synthesise a URL just to feed it to the parser.
 */
export function extractYouTubeVideoId(input: string | null | undefined): string {
  if (!input) return '';
  const trimmed = input.trim();
  if (!trimmed) return '';

  // Bare video ID pass-through. YouTube IDs are 11 chars from [A-Za-z0-9_-].
  // We accept the bare ID rather than parsing a synthetic URL because the
  // calendar/live endpoint may eventually ship just the ID (issue #283).
  if (/^[A-Za-z0-9_-]{11}$/.test(trimmed)) {
    return trimmed;
  }

  // youtu.be/<id> — short URL form.
  const shortMatch = trimmed.match(/youtu\.be\/([A-Za-z0-9_-]{11})/);
  if (shortMatch) return shortMatch[1];

  // youtube.com/watch?v=<id> + youtube.com/embed/<id> + youtube.com/shorts/<id>.
  // Match on any subdomain (www., m., none) and accept both http and https.
  const longMatch = trimmed.match(
    /(?:youtube\.com|youtube-nocookie\.com)\/(?:watch\?v=|embed\/|shorts\/|live\/)([A-Za-z0-9_-]{11})/,
  );
  if (longMatch) return longMatch[1];

  // As a last resort, look for a `v=` query parameter anywhere in the URL
  // (covers edge cases like youtube.com/watch?feature=shared&v=<id>).
  try {
    const url = new URL(trimmed);
    const v = url.searchParams.get('v');
    if (v && /^[A-Za-z0-9_-]{11}$/.test(v)) return v;
  } catch {
    // Not a URL — fall through to the empty return.
  }
  return '';
}

/**
 * YouTubeEmbed renders a responsive 16:9 YouTube iframe player.
 *
 * The component renders NOTHING when `videoId` is falsy — callers can
 * always render `<YouTubeEmbed videoId={bill.video_url ? extractYouTubeVideoId(bill.video_url) : ''} />`
 * without an extra null guard. This keeps the Bill detail page markup
 * flat when the Bill has no Hansard video yet.
 */
export function YouTubeEmbed({
  videoId,
  title = 'YouTube video player',
  className,
  autoplay = false,
  showWatchLink = true,
}: YouTubeEmbedProps) {
  if (!videoId) return null;

  // Build the embed URL. The privacy-enhanced nocookie domain is used so
  // YouTube does not set tracking cookies until the user actually plays
  // the video. The `rel=0` parameter limits the "related videos" shown at
  // the end of the playback to the same channel (the parliament's), and
  // `modestbranding=1` reduces the YouTube logo overlay.
  const params = new URLSearchParams({
    rel: '0',
    modestbranding: '1',
  });
  if (autoplay) params.set('autoplay', '1');
  const embedSrc = `${youTubeNoCookieDomain}/embed/${encodeURIComponent(videoId)}?${params.toString()}`;
  const watchHref = `https://www.youtube.com/watch?v=${encodeURIComponent(videoId)}`;

  return (
    <figure className={className}>
      <div className="relative aspect-video w-full overflow-hidden rounded-lg border border-civic-border bg-civic-ink">
        <iframe
          src={embedSrc}
          title={title}
          loading="lazy"
          allow={youTubeEmbedAllow}
          allowFullScreen
          referrerPolicy="strict-origin-when-cross-origin"
          className="absolute inset-0 h-full w-full"
          // The iframe needs a sandbox-free cross-origin policy so
          // YouTube can run its player script; the allow attribute
          // already restricts the API surface.
        />
      </div>
      {showWatchLink && (
        <figcaption className="mt-2 flex items-center justify-between text-xs text-civic-stone">
          <a
            href={watchHref}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-1 font-medium text-civic-leaf hover:underline"
          >
            Watch on YouTube
            <ExternalLink className="h-3 w-3" aria-hidden="true" />
          </a>
          <span className="text-civic-stone/70">Streamed via parliament.go.ke</span>
        </figcaption>
      )}
    </figure>
  );
}

export default YouTubeEmbed;
