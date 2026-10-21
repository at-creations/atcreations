import { getLocale } from '$lib/paraglide/runtime';
import { isLocale, DEFAULT_LOCALE, type Locale } from './i18n';

/** Active locale (from the URL on the client, from the request on the server). */
export function currentLocale(): Locale {
	const locale = getLocale();
	return isLocale(locale) ? locale : DEFAULT_LOCALE;
}
