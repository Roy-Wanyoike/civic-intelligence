'use client';

import { Moon, Sun } from 'lucide-react';
import { useTheme } from './theme-provider';

export function ThemeToggle() {
  const { theme, toggleTheme } = useTheme();
  return (
    <button
      type="button"
      onClick={toggleTheme}
      className="rounded-lg p-2 text-civic-stone hover:bg-civic-mist hover:text-civic-leaf transition"
      aria-label={theme === 'light' ? 'Switch to dark mode' : 'Switch to light mode'}
    >
      {theme === 'light' ? (
        <Moon className="h-5 w-5" aria-hidden="true" />
      ) : (
        <Sun className="h-5 w-5" aria-hidden="true" />
      )}
    </button>
  );
}
