/**
 * Civic Intelligence — Design Tokens
 * -----------------------------------
 * Single source of truth for the platform's visual language. Tailwind
 * config (`tailwind.config.ts`) imports these tokens so utility classes
 * like `bg-civic-forest` and `text-civic-stone` stay in sync with the
 * canonical values declared here.
 *
 * Use these tokens directly when you need a literal value outside Tailwind
 * (inline styles, JS contexts like `metadata.themeColor`, canvas, etc.).
 *
 * DO NOT introduce new colors/spacing/typography ad-hoc. Add them here first.
 */

export const colors = {
  forest: '#0c3b2e',
  leaf: '#15784a',
  acacia: '#d4a017',
  clay: '#9c4221',
  stone: '#52606b',
  mist: '#f5f7f6',
  ink: '#0b1f17',
  paper: '#ffffff',
  border: '#c8d3cc',
} as const;

export const spacing = {
  xs: '0.25rem', // 4px
  sm: '0.5rem', // 8px
  md: '0.75rem', // 12px
  base: '1rem', // 16px
  lg: '1.5rem', // 24px
  xl: '2rem', // 32px
  '2xl': '3rem', // 48px
  '3xl': '4rem', // 64px
} as const;

export const typography = {
  xs: '0.75rem',
  sm: '0.875rem',
  base: '1rem',
  lg: '1.125rem',
  xl: '1.25rem',
  '2xl': '1.5rem',
  '3xl': '1.875rem',
  '4xl': '2.25rem',
} as const;

export const radius = {
  sm: '0.25rem',
  md: '0.5rem',
  lg: '0.75rem',
  full: '9999px',
} as const;

export const shadow = {
  sm: '0 1px 2px rgba(0,0,0,0.05)',
  md: '0 4px 6px rgba(0,0,0,0.07)',
  lg: '0 10px 15px rgba(0,0,0,0.1)',
} as const;

/** Convenience aggregate for documentation, introspection, and JSON serialization. */
export const tokens = {
  colors,
  spacing,
  typography,
  radius,
  shadow,
} as const;

/**
 * Confidence-pill background colors — used by the `.pill-*` classes in
 * `globals.css` and the `confidenceClass()` helper in `lib/utils.ts`.
 * The `unknown` level falls back to `stone` (no bespoke gray).
 */
export const confidenceColors = {
  high: colors.leaf,
  medium: colors.acacia,
  low: colors.stone,
  unknown: colors.stone,
} as const;
