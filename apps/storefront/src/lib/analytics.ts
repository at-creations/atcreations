/**
 * First-party, cookieless page-view beacon (see POST /v1/events/pageview). Sends only the path
 * and, on the first page of a visit, the referrer. Nothing is stored in the browser.
 */

/** Browser opt-out signals: Do Not Track and Global Privacy Control. */
export function optedOut(nav: Navigator = navigator): boolean {
	const n = nav as Navigator & { globalPrivacyControl?: boolean };
	return n.doNotTrack === '1' || n.globalPrivacyControl === true;
}

export function pageViewBody(path: string, referrer: string): string {
	return JSON.stringify(referrer ? { path, referrer } : { path });
}

export function trackPageView(apiUrl: string, path: string, referrer = ''): void {
	if (!apiUrl || optedOut()) return;
	// Fire-and-forget; analytics must never affect the page. keepalive lets it finish on unload.
	fetch(`${apiUrl.replace(/\/+$/, '')}/v1/events/pageview`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: pageViewBody(path, referrer),
		credentials: 'omit',
		keepalive: true,
		mode: 'cors'
	}).catch(() => {});
}
