import { describe, expect, it } from 'vitest';
import en from '../../messages/en.json';
import vi from '../../messages/vi.json';

/** A plain string, or variants selected by a matcher (e.g. plural forms). */
type Message = string | { match: Record<string, string> }[];

const texts = (message: Message | undefined): string[] =>
	message === undefined
		? []
		: typeof message === 'string'
			? [message]
			: message.flatMap((variant) => Object.values(variant.match));

const params = (message: Message | undefined) =>
	[
		...new Set(texts(message).flatMap((s) => [...s.matchAll(/\{(\w+)\}/g)].map((m) => m[1])))
	].sort();

describe('message catalogs', () => {
	it('have the same keys in en and vi', () => {
		expect(Object.keys(vi).sort()).toEqual(Object.keys(en).sort());
	});
	it('use the same parameters in both locales', () => {
		const viMap = vi as unknown as Record<string, Message>;
		for (const [key, value] of Object.entries(en as unknown as Record<string, Message>)) {
			if (key === '$schema') continue;
			expect(params(viMap[key]), key).toEqual(params(value));
		}
	});
});
