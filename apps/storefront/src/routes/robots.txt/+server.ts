import type { RequestHandler } from './$types';
import { config } from '$lib/server/env';

export const GET: RequestHandler = () => {
	const { siteUrl } = config();
	const body = ['User-agent: *', 'Allow: /', '', `Sitemap: ${siteUrl}/sitemap.xml`, ''].join('\n');
	return new Response(body, {
		headers: {
			'content-type': 'text/plain; charset=utf-8',
			'cache-control': 'public, max-age=86400'
		}
	});
};
