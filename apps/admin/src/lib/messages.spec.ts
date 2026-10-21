import { describe, expect, it } from 'vitest';
import en from '../../messages/en.json';
import vi from '../../messages/vi.json';

const params = (s: string) => [...s.matchAll(/\{(\w+)\}/g)].map((m) => m[1]).sort();

describe('message catalogs', () => {
	it('have the same keys in en and vi', () => {
		expect(Object.keys(vi).sort()).toEqual(Object.keys(en).sort());
	});
	it('use the same parameters in both locales', () => {
		const viMap = vi as Record<string, string>;
		for (const [key, value] of Object.entries(en as Record<string, string>)) {
			if (key === '$schema') continue;
			expect(params(viMap[key] ?? ''), key).toEqual(params(value));
		}
	});
});
