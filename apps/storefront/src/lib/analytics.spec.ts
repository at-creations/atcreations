import { describe, expect, it } from 'vitest';
import { optedOut, pageViewBody } from './analytics';

describe('optedOut', () => {
	it('honours Do Not Track and Global Privacy Control', () => {
		expect(optedOut({ doNotTrack: '1' } as Navigator)).toBe(true);
		expect(optedOut({ globalPrivacyControl: true } as unknown as Navigator)).toBe(true);
		expect(optedOut({ doNotTrack: null } as Navigator)).toBe(false);
	});
});

describe('pageViewBody', () => {
	it('sends the referrer only when there is one', () => {
		expect(JSON.parse(pageViewBody('/en', ''))).toEqual({ path: '/en' });
		expect(JSON.parse(pageViewBody('/en', 'https://google.com/'))).toEqual({
			path: '/en',
			referrer: 'https://google.com/'
		});
	});
});
