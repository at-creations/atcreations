import { intlLocale, type Locale } from './i18n';

const moneyFormatters = new Map<Locale, Intl.NumberFormat>();

/** Format integer cents (CAD) for display. */
export function formatMoney(cents: number, locale: Locale): string {
	let formatter = moneyFormatters.get(locale);
	if (!formatter) {
		formatter = new Intl.NumberFormat(intlLocale(locale), { style: 'currency', currency: 'CAD' });
		moneyFormatters.set(locale, formatter);
	}
	return formatter.format(cents / 100);
}

/** Format an `HH:MM` 24h time string for display in the given locale. */
export function formatTime(hhmm: string, locale: Locale): string {
	const match = /^(\d{2}):(\d{2})$/.exec(hhmm);
	if (!match) return hhmm;
	const date = new Date(Date.UTC(2000, 0, 1, Number(match[1]), Number(match[2])));
	return new Intl.DateTimeFormat(intlLocale(locale), {
		hour: 'numeric',
		minute: '2-digit',
		timeZone: 'UTC'
	}).format(date);
}
