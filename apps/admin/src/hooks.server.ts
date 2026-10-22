import { redirect, type Handle } from '@sveltejs/kit';
import { sequence } from '@sveltejs/kit/hooks';
import { withClerkHandler } from 'svelte-clerk/server';
import { getTextDirection } from '$lib/paraglide/runtime.js';
import { paraglideMiddleware } from '$lib/paraglide/server.js';
import { safeRedirectPath } from '$lib/utils/redirect';

const PUBLIC_PREFIX = '/sign-in';
/** Routes reachable without a session: sign-in and invitation sign-up. */
const PUBLIC_PREFIXES = [PUBLIC_PREFIX, '/sign-up'];

function isPublicPath(pathname: string): boolean {
	return PUBLIC_PREFIXES.some((p) => pathname === p || pathname.startsWith(`${p}/`));
}

/** Every route except /sign-in/* and /sign-up/* requires a Clerk session. Roles are checked by the API. */
const authGuard: Handle = async ({ event, resolve }) => {
	const { pathname, search } = event.url;
	if (!isPublicPath(pathname)) {
		const { userId } = event.locals.auth();
		if (!userId) {
			const target = safeRedirectPath(`${pathname}${search}`);
			const query = target === '/' ? '' : `?redirect_url=${encodeURIComponent(target)}`;
			redirect(303, `${PUBLIC_PREFIX}${query}`);
		}
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

export const handle: Handle = sequence(withClerkHandler(), authGuard, securityHeaders, i18n);
