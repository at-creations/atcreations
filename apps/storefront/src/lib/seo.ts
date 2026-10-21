/**
 * Serialize data for an inline `<script type="application/ld+json">` block.
 * `<`, `>`, `&` and the JS line separators are escaped so the payload can never
 * terminate the script element or be interpreted as HTML.
 */
export function serializeJsonLd(data: unknown): string {
	return JSON.stringify(data)
		.replace(/</g, '\\u003c')
		.replace(/>/g, '\\u003e')
		.replace(/&/g, '\\u0026')
		.replace(/\u2028/g, '\\u2028')
		.replace(/\u2029/g, '\\u2029');
}

/** Collapse whitespace and truncate text for meta descriptions without cutting words. */
export function truncate(text: string, max = 160): string {
	const clean = text.replace(/\s+/g, ' ').trim();
	if (clean.length <= max) return clean;
	const cut = clean.slice(0, max - 1);
	const lastSpace = cut.lastIndexOf(' ');
	return `${(lastSpace > max * 0.6 ? cut.slice(0, lastSpace) : cut).trimEnd()}…`;
}
