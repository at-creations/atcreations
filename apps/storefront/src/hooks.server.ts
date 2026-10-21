import { building } from '$app/environment';
import type { Handle, HandleServerError } from '@sveltejs/kit';
import { sequence } from '@sveltejs/kit/hooks';
import { getTextDirection, shouldRedirect } from '$lib/paraglide/runtime';
import { paraglideMiddleware } from '$lib/paraglide/server';
import { config } from '$lib/server/env';
import { runWithRequestContext } from '$lib/server/request-context';

// Validate environment at startup: a misconfigured server fails fast instead of serving errors.
if (!building) config();

const SECURITY_HEADERS: Record<string, string> = {
	'X-Content-Type-Options': 'nosniff',
	'Referrer-Policy': 'strict-origin-when-cross-origin',
	'Permissions-Policy': 'camera=(), microphone=(), geolocation=()',
	'Cross-Origin-Opener-Policy': 'same-origin',
	'X-Frame-Options': 'DENY'
	// HSTS is set by the reverse proxy.
};

const securityHeaders: Handle = async ({ event, resolve }) => {
	const response = await resolve(event);
	for (const [name, value] of Object.entries(SECURITY_HEADERS)) {
		try {
			if (!response.headers.has(name)) response.headers.set(name, value);
		} catch {
			// Some responses (e.g. Response.redirect) have immutable headers.
		}
	}
	return response;
};

/** Paths served without a locale prefix. */
const UNLOCALIZED = new Set(['/sitemap.xml', '/robots.txt']);
const LOCALE_PREFIX = /^\/(en|vi)(\/|$)/;

/**
 * Any GET to a path without a locale prefix (including `/`) is redirected to its localized
 * equivalent, choosing the locale from the cookie set by the locale switcher, then
 * Accept-Language, then English. Unlike paraglide's built-in redirect, this also applies to
 * non-document requests so that crawlers and curl get a redirect too.
 */
const localeRedirect: Handle = async ({ event, resolve }) => {
	const { pathname, search } = event.url;
	const method = event.request.method;
	if (
		(method === 'GET' || method === 'HEAD') &&
		!LOCALE_PREFIX.test(pathname) &&
		!UNLOCALIZED.has(pathname)
	) {
		const decision = await shouldRedirect({ request: event.request });
		if (decision.shouldRedirect && decision.redirectUrl) {
			// Canonical URLs have no trailing slash (`/en`, not `/en/`).
			const target = decision.redirectUrl.pathname.replace(/(.)\/+$/, '$1');
			return new Response(null, {
				status: 307,
				headers: {
					Location: target + search,
					Vary: 'Accept-Language, Cookie',
					'Cache-Control': 'private, no-store'
				}
			});
		}
	}
	return resolve(event);
};

const handleParaglide: Handle = ({ event, resolve }) =>
	paraglideMiddleware(event.request, ({ request, locale }) => {
		return resolve(
			{ ...event, request },
			{
				transformPageChunk: ({ html }) =>
					html
						.replace('%paraglide.lang%', locale)
						.replace('%paraglide.dir%', getTextDirection(locale))
			}
		);
	});

/** Makes the visitor's address available to server-side API calls. */
const requestContext: Handle = ({ event, resolve }) => {
	let clientAddress: string | undefined;
	try {
		clientAddress = event.getClientAddress();
	} catch {
		// Not available during prerendering.
	}
	return runWithRequestContext({ clientAddress }, () => resolve(event));
};

export const handle: Handle = sequence(
	requestContext,
	securityHeaders,
	localeRedirect,
	handleParaglide
);

export const handleError: HandleServerError = ({ error, status }) => {
	if (status !== 404) {
		console.error('[storefront] unhandled error', error instanceof Error ? error.stack : error);
	}
	// Never leak internals to the client.
	return { message: status === 404 ? 'Not found' : 'Internal error' };
};
