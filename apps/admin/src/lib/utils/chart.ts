/** Helpers shared by the SVG charts. */

/**
 * Round axis ticks from 0 up to at least `max`: steps of 1, 2 or 5 × 10^n, about `count` of them.
 * An all-zero series still gets a 0–1 axis so the plot has height.
 */
export function niceTicks(max: number, count = 4): number[] {
	if (!Number.isFinite(max) || max <= 0) return [0, 1];
	const raw = max / count;
	const power = 10 ** Math.floor(Math.log10(raw));
	const step = [1, 2, 5, 10].map((m) => m * power).find((s) => s >= raw) ?? 10 * power;
	const ticks: number[] = [];
	for (let v = 0; v < max + step / 2; v += step) ticks.push(Number(v.toPrecision(12)));
	if (ticks[ticks.length - 1]! < max) ticks.push(ticks[ticks.length - 1]! + step);
	return ticks;
}

const compactFormat = new Map<string, Intl.NumberFormat>();

/** 950 → "950", 1 284 → "1.3K", 4 200 000 → "4.2M". */
export function compactNumber(n: number, locale = 'en-CA'): string {
	let f = compactFormat.get(locale);
	if (!f) {
		f = new Intl.NumberFormat(locale, { notation: 'compact', maximumFractionDigits: 1 });
		compactFormat.set(locale, f);
	}
	return f.format(n);
}

/**
 * Relative change from `previous` to `current`, as a fraction (0.25 = +25 %).
 * null when there is no baseline to compare against.
 */
export function change(current: number, previous: number): number | null {
	if (previous === 0) return null;
	return (current - previous) / previous;
}
