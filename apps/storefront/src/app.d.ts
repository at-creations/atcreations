// See https://svelte.dev/docs/kit/types#app.d.ts
import type { StoreInfo } from '@atcreations/api-types';
import type { Locale } from '$lib/i18n';

declare global {
	namespace App {
		interface Error {
			message: string;
		}
		interface PageData {
			locale: Locale;
			config: {
				siteUrl: string;
				publicApiUrl: string;
				assetBaseUrl: string;
			};
			storeInfo: StoreInfo | null;
		}
		// interface Locals {}
		// interface PageState {}
		// interface Platform {}
	}
}

export {};
