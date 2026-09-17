import type { Config } from 'tailwindcss';
import { colors } from './src/lib/design-tokens';

const config: Config = {
  content: ['./src/**/*.{ts,tsx,mdx}'],
  theme: {
    extend: {
      colors: {
        // Single source of truth lives in src/lib/design-tokens.ts.
        // Kenya-inspired but politically neutral palette: independence green
        // + acacia + savanna gold, kept muted for accessibility.
        civic: {
          ink: colors.ink,
          forest: colors.forest,
          leaf: colors.leaf,
          acacia: colors.acacia,
          clay: colors.clay,
          stone: colors.stone,
          mist: colors.mist,
          paper: colors.paper,
          border: colors.border,
        },
      },
      fontFamily: {
        sans: ['var(--font-inter)', 'system-ui', 'sans-serif'],
        serif: ['var(--font-serif)', 'Georgia', 'serif'],
      },
      typography: {
        DEFAULT: {
          css: {
            maxWidth: '70ch',
          },
        },
      },
    },
  },
  plugins: [],
};

export default config;
