import { redirect, type Handle } from '@sveltejs/kit';
import { sequence } from '@sveltejs/kit/hooks';
import { withClerkHandler } from 'svelte-clerk/server';
import { cookieMaxAge, cookieName, getTextDirection, locales } from '$lib/paraglide/runtime.js';
import { paraglideMiddleware } from '$lib/paraglide/server.js';
import { safeRedirectPath } from '$lib/utils/redirect';

const PUBLIC_PREFIX = '/sign-in';
/** Routes reachable without a session: sign-in and invitation sign-up. */
const PUBLIC_PREFIXES = [PUBLIC_PREFIX, '/sign-up'];

function isPublicPath(pathname: string): boolean {
	return PUBLIC_PREFIXES.some((p) => pathname === p || pathname.startsWith(`${p}/`));
}

/**
 * Every route except /sign-in/* and /sign-up/* requires a Clerk session; signed-in users are sent
 * past /sign-in. Roles are checked by the API.
 */
const authGuard: Handle = async ({ event, resolve }) => {
	const { pathname, search } = event.url;
	if (pathname === PUBLIC_PREFIX || pathname.startsWith(`${PUBLIC_PREFIX}/`)) {
		// Already signed in: skip the sign-in form and continue where the user was headed.
		if (event.locals.auth().userId) {
			redirect(303, safeRedirectPath(event.url.searchParams.get('redirect_url')));
		}
	} else if (!isPublicPath(pathname)) {
		const { userId } = event.locals.auth();
		if (!userId) {
			const target = safeRedirectPath(`${pathname}${search}`);
			const query = target === '/' ? '' : `?redirect_url=${encodeURIComponent(target)}`;
			redirect(303, `${PUBLIC_PREFIX}${query}`);
		}
	}
	return resolve(event);
};

const LOCALE_PREFIX = new RegExp(`^/(${locales.join('|')})(?=/|$)`);

/**
 * Admin URLs carry no locale (it lives in the Paraglide cookie). Links shaped like the
 * storefront's (/en/..., /vi/...) switch the locale and redirect to the unprefixed path.
 */
const localePrefix: Handle = async ({ event, resolve }) => {
	const match = LOCALE_PREFIX.exec(event.url.pathname);
	const locale = match?.[1];
	if (match && locale && (event.request.method === 'GET' || event.request.method === 'HEAD')) {
		event.cookies.set(cookieName, locale, {
			path: '/',
			maxAge: cookieMaxAge,
			httpOnly: false, // read by the Paraglide client runtime
			sameSite: 'lax'
		});
		let rest = event.url.pathname.slice(match[0].length) || '/';
		if (rest.startsWith('//')) rest = '/'; // never a protocol-relative redirect
		redirect(302, `${rest}${event.url.search}`);
	}
	return resolve(event);
};

const i18n: Handle = ({ event, resolve }) =>
	paraglideMiddleware(event.request, ({ request, locale }) => {
		event.request = request;
		return resolve(event, {
			transformPageChunk: ({ html }) =>
				html
					.replace('%paraglide.lang%', locale)
					.replace('%paraglide.dir%', getTextDirection(locale))
		});
	});

const securityHeaders: Handle = async ({ event, resolve }) => {
	const response = await resolve(event);
	const h = response.headers;
	h.set('X-Content-Type-Options', 'nosniff');
	h.set('Referrer-Policy', 'same-origin');
	h.set(
		'Permissions-Policy',
		'camera=(), microphone=(), geolocation=(), payment=(), usb=(), interest-cohort=()'
	);
	h.set('Cross-Origin-Opener-Policy', 'same-origin');
	h.set('X-Frame-Options', 'DENY');
	h.set('X-Robots-Tag', 'noindex, nofollow');
	if (h.get('content-type')?.startsWith('text/html')) h.set('Cache-Control', 'no-store');
	return response;
};

export const handle: Handle = sequence(
	localePrefix,
	withClerkHandler(),
	authGuard,
	securityHeaders,
	i18n
);
