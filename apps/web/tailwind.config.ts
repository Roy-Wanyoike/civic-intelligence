import type { Config } from 'tailwindcss';

const config: Config = {
  content: ['./src/**/*.{ts,tsx,mdx}'],
  theme: {
    extend: {
      colors: {
        // Kenya-inspired but politically neutral palette.
        // Independence green + acacia + savanna gold, kept muted for accessibility.
        civic: {
          ink: '#0b1f17',
          forest: '#0c3b2e',
          leaf: '#15784a',
          acacia: '#d4a017',
          clay: '#9c4221',
          stone: '#52606b',
          mist: '#f5f7f6',
          paper: '#ffffff',
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
