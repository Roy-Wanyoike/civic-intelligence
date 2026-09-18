'use client';

/**
 * Language switcher (spec §66).
 *
 * Renders a globe icon button that toggles a dropdown listing supported
 * locales (English, Kiswahili). Selecting an option persists the choice
 * in the `civic-locale` cookie (read by `src/i18n/request.ts`) and reloads
 * the page so the new locale takes effect.
 *
 * NOTE: This is a non-routed i18n setup — there is no `[locale]` URL
 * segment, so a hard reload is the simplest way to re-render with the
 * new message bundle. When the app migrates to routed i18n, replace
 * `window.location.reload()` with `router.replace(\`/${locale}${pathname}\`)`.
 */
import { useEffect, useRef, useState } from 'react';
import { Globe, Check, ChevronDown } from 'lucide-react';
import { useLocale, useTranslations } from 'next-intl';

const LOCALE_OPTIONS = [
  { code: 'en', labelKey: 'english' as const },
  { code: 'sw', labelKey: 'kiswahili' as const },
] as const;

const COOKIE_NAME = 'civic-locale';
const COOKIE_MAX_AGE = 60 * 60 * 24 * 365; // 1 year

export function LanguageSwitcher() {
  const t = useTranslations('language');
  const locale = useLocale();
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  // Close on outside click or Escape.
  useEffect(() => {
    if (!open) return;
    function handleClickOutside(e: MouseEvent) {
      if (
        containerRef.current &&
        !containerRef.current.contains(e.target as Node)
      ) {
        setOpen(false);
      }
    }
    function handleEscape(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false);
    }
    document.addEventListener('mousedown', handleClickOutside);
    document.addEventListener('keydown', handleEscape);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleEscape);
    };
  }, [open]);

  function selectLocale(code: string) {
    // Persist via cookie so the server-side `getRequestConfig` can pick it up.
    document.cookie = `${COOKIE_NAME}=${code}; path=/; max-age=${COOKIE_MAX_AGE}; SameSite=Lax`;
    setOpen(false);
    // Reload so server components re-read messages for the new locale.
    window.location.reload();
  }

  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        aria-haspopup="listbox"
        aria-label={t('label')}
        title={t('label')}
        className="inline-flex h-9 w-9 items-center justify-center rounded-full text-civic-forest hover:bg-civic-mist"
      >
        <Globe className="h-5 w-5" aria-hidden="true" />
        <span className="sr-only">{t('label')}</span>
      </button>

      {open && (
        <div
          role="listbox"
          aria-label={t('label')}
          className="absolute right-0 top-full z-50 mt-1 w-44 overflow-hidden rounded-lg border border-civic-border bg-civic-paper shadow-xl"
        >
          <div className="flex items-center justify-between border-b border-civic-border px-3 py-2">
            <span className="text-[11px] font-semibold uppercase tracking-wider text-civic-stone">
              {t('label')}
            </span>
            <ChevronDown className="h-3 w-3 text-civic-stone" aria-hidden="true" />
          </div>
          <ul className="py-1">
            {LOCALE_OPTIONS.map((opt) => {
              const isActive = opt.code === locale;
              return (
                <li key={opt.code}>
                  <button
                    type="button"
                    role="option"
                    aria-selected={isActive}
                    onClick={() => selectLocale(opt.code)}
                    className="flex w-full items-center justify-between px-3 py-2 text-sm text-civic-ink hover:bg-civic-mist hover:text-civic-leaf"
                  >
                    <span>{t(opt.labelKey)}</span>
                    {isActive && (
                      <Check
                        className="h-4 w-4 text-civic-leaf"
                        aria-hidden="true"
                      />
                    )}
                  </button>
                </li>
              );
            })}
          </ul>
          <div className="border-t border-civic-border px-3 py-1.5 text-[10px] text-civic-stone">
            <span className="font-mono">{locale.toUpperCase()}</span> · current
          </div>
        </div>
      )}
    </div>
  );
}
