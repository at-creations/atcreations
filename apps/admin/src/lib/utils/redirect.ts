/**
 * Validate a post-sign-in redirect target. Only same-origin relative paths that start with a
 * single "/" are allowed; anything else (protocol-relative "//", backslashes, schemes, control
 * characters, malformed encodings) falls back to `fallback`.
 */
export function safeRedirectPath(input: string | null | undefined, fallback = '/'): string {
	if (typeof input !== 'string' || input.length === 0 || input.length > 2048) return fallback;
	if (!input.startsWith('/')) return fallback;
	if (input.startsWith('//') || input.includes('\\')) return fallback;
	for (let i = 0; i < input.length; i++) {
		const code = input.charCodeAt(i);
		if (code < 0x20 || code === 0x7f) return fallback;
	}
	let decoded: string;
	try {
		decoded = decodeURIComponent(input);
	} catch {
		return fallback;
	}
	if (decoded.startsWith('//') || decoded.includes('\\')) return fallback;
	const base = 'http://admin.invalid';
	let url: URL;
	try {
		url = new URL(input, base);
	} catch {
		return fallback;
	}
	if (url.origin !== base) return fallback;
	// Dot segments are resolved by the parser ("/.//evil.com" → "//evil.com"), so check the result too.
	if (url.pathname.startsWith('//')) return fallback;
	const result = `${url.pathname}${url.search}${url.hash}`;
	if (url.pathname === '/sign-in' || url.pathname.startsWith('/sign-in/')) return fallback;
	return result;
}
