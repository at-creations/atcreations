import type { LayoutServerLoad } from './$types';
import { getStoreInfo, orFallback } from '$lib/server/api';
import { config } from '$lib/server/env';
import { currentLocale } from '$lib/locale';

export const load: LayoutServerLoad = async () => {
	const { siteUrl, publicApiUrl, assetBaseUrl } = config();
	// Store info powers the footer and contact sections; the site still renders without it.
	const storeInfo = await orFallback(getStoreInfo, null);
	return {
		locale: currentLocale(),
		config: { siteUrl, publicApiUrl, assetBaseUrl },
		storeInfo
	};
};
