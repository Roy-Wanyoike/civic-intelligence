/**
 * next-intl request configuration — locale resolver.
 *
 * Spec §66 — Internationalization readiness. The platform is primarily
 * English-first for Kenya, but the architecture must support Swahili
 * (and future African languages) without code changes.
 *
 * Strategy (non-routed i18n):
 * - No `[locale]` dynamic segment is used. The whole app stays at `/...`
 *   and the active locale is selected per-request from the `civic-locale`
 *   cookie (set by the language switcher in the Header).
 * - Defaults to `'en'` when no cookie is present or the value is unknown.
 * - Messages are loaded synchronously via static imports so the bundler
 *   can tree-shake per locale.
 *
 * Future enhancement: when adding `[locale]` routing, replace the manual
 * cookie lookup with the `requestLocale` parameter provided by next-intl.
 *
 * @see https://next-intl.dev/docs/getting-started/app-router/without-i18n-routing
 */
import { getRequestConfig } from 'next-intl/server';
import { cookies } from 'next/headers';

export const locales = ['en', 'sw'] as const;
export type AppLocale = (typeof locales)[number];
export const defaultLocale: AppLocale = 'en';

export default getRequestConfig(async () => {
  // Read the locale from the `civic-locale` cookie. Falls back to 'en'.
  const cookieStore = await cookies();
  const cookieValue = cookieStore.get('civic-locale')?.value;
  const locale: AppLocale =
    cookieValue && locales.includes(cookieValue as AppLocale)
      ? (cookieValue as AppLocale)
      : defaultLocale;

  const messages = (await import(`./locales/${locale}.json`)).default;

  return {
    locale,
    messages,
    // Use the user's locale for default date/number formatting.
    timeZone: 'Africa/Nairobi',
  };
});
