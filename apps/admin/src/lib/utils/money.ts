/** Money helpers. The API stores CAD as integer cents. */

export type ParsedDollars =
	| { ok: true; cents: number | null }
	| { ok: false; reason: 'invalid' | 'negative' | 'precision' | 'too_large' };

export const MAX_PRICE_CENTS = 100_000_000;

/**
 * Parse a user-typed dollar amount ("19.99", "$1,234.5", "") into integer cents.
 * Empty input yields `cents: null` (e.g. "contact for price").
 */
export function parseDollars(input: string | number | null | undefined): ParsedDollars {
	if (input === null || input === undefined) return { ok: true, cents: null };
	const raw = String(input).trim().replace(/^\$/, '').replace(/,/g, '').trim();
	if (raw === '') return { ok: true, cents: null };
	if (raw.startsWith('-')) return { ok: false, reason: 'negative' };
	if (!/^\d+(\.\d*)?$|^\.\d+$/.test(raw)) return { ok: false, reason: 'invalid' };
	const [whole = '0', frac = ''] = raw.split('.');
	if (frac.length > 2) return { ok: false, reason: 'precision' };
	const cents = Number(whole || '0') * 100 + Number(frac.padEnd(2, '0') || '0');
	if (!Number.isSafeInteger(cents) || cents > MAX_PRICE_CENTS) {
		return { ok: false, reason: 'too_large' };
	}
	return { ok: true, cents };
}

/** Convert cents to a plain dollar string for inputs (1999 → "19.99"); null → "". */
export function centsToDollarInput(cents: number | null | undefined): string {
	if (cents === null || cents === undefined) return '';
	const sign = cents < 0 ? '-' : '';
	const abs = Math.abs(Math.trunc(cents));
	return `${sign}${Math.floor(abs / 100)}.${String(abs % 100).padStart(2, '0')}`;
}

const cadFormatter = new Intl.NumberFormat('en-CA', { style: 'currency', currency: 'CAD' });

/** Format cents as CAD using the en-CA locale ("$19.99"). */
export function formatCAD(cents: number): string {
	return cadFormatter.format(cents / 100);
}
