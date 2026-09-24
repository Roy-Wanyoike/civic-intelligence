import type { Guide, GuideCategory } from './types';
import { howABillBecomesLaw } from './how-a-bill-becomes-law';
import { understandingParliament } from './understanding-parliament';
import { howToReadABill } from './how-to-read-a-bill';
import { yourConstitutionalRights } from './your-constitutional-rights';
import { publicParticipation } from './public-participation';

/**
 * Registry of all published civic-education guides.
 *
 * Adding a 6th guide:
 *   1. Create a new file `guides/<slug>.tsx` exporting a `Guide` object.
 *   2. Import it here and append to the `guides` array.
 *   3. Add a category colour (if a new category) to `CATEGORY_STYLES`.
 *
 * The renderer at `learn/[slug]/page.tsx` pre-renders every slug in this
 * array via `generateStaticParams`. The landing page at `learn/page.tsx`
 * renders one card per guide.
 *
 * Slugs must be unique and URL-safe (lowercase, hyphen-separated). They
 * form the URL `https://<site>/learn/<slug>` and are also used as the
 * SEO canonical URL fragment.
 */
export const guides: Guide[] = [
  howABillBecomesLaw,
  understandingParliament,
  howToReadABill,
  yourConstitutionalRights,
  publicParticipation,
];

/** Quick slug → guide lookup used by the dynamic route. */
export function getGuideBySlug(slug: string): Guide | undefined {
  return guides.find((g) => g.slug === slug);
}

/** All slugs — used by generateStaticParams. */
export function getAllGuideSlugs(): string[] {
  return guides.map((g) => g.slug);
}

/**
 * Per-category accent colours. Each guide card on the landing page is
 * colour-coded by category so a reader can scan the grid and find what
 * they are looking for by hue.
 *
 * The classes are civic-* tokens so dark-mode overrides in globals.css
 * apply automatically. The mapping is exhaustive over `GuideCategory`.
 */
export const CATEGORY_STYLES: Record<
  GuideCategory,
  { badge: string; dot: string; ring: string }
> = {
  Legislation: {
    badge: 'bg-civic-leaf/10 text-civic-leaf border-civic-leaf/30',
    dot: 'bg-civic-leaf',
    ring: 'hover:border-civic-leaf/40',
  },
  Parliament: {
    badge: 'bg-civic-acacia/20 text-civic-ink border-civic-acacia/40',
    dot: 'bg-civic-acacia',
    ring: 'hover:border-civic-acacia/50',
  },
  Constitution: {
    badge: 'bg-civic-forest/10 text-civic-forest border-civic-forest/30',
    dot: 'bg-civic-forest',
    ring: 'hover:border-civic-forest/40',
  },
  'Civic Participation': {
    badge: 'bg-civic-clay/10 text-civic-clay border-civic-clay/30',
    dot: 'bg-civic-clay',
    ring: 'hover:border-civic-clay/40',
  },
  'Reading Skills': {
    badge: 'bg-civic-stone/15 text-civic-ink border-civic-border',
    dot: 'bg-civic-stone',
    ring: 'hover:border-civic-stone/50',
  },
};
