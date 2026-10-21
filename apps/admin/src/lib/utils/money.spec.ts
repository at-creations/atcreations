import { describe, expect, it } from 'vitest';
import { centsToDollarInput, formatCAD, parseDollars } from './money';

describe('parseDollars', () => {
	it('parses whole and decimal amounts into cents', () => {
		expect(parseDollars('19.99')).toEqual({ ok: true, cents: 1999 });
		expect(parseDollars('5')).toEqual({ ok: true, cents: 500 });
		expect(parseDollars('5.5')).toEqual({ ok: true, cents: 550 });
		expect(parseDollars('.25')).toEqual({ ok: true, cents: 25 });
		expect(parseDollars('$1,234.50')).toEqual({ ok: true, cents: 123450 });
		expect(parseDollars(0.1)).toEqual({ ok: true, cents: 10 });
	});
	it('avoids floating point drift', () => {
		expect(parseDollars('0.29')).toEqual({ ok: true, cents: 29 });
		expect(parseDollars('1.15')).toEqual({ ok: true, cents: 115 });
	});
	it('treats empty input as null', () => {
		expect(parseDollars('')).toEqual({ ok: true, cents: null });
		expect(parseDollars('   ')).toEqual({ ok: true, cents: null });
		expect(parseDollars(null)).toEqual({ ok: true, cents: null });
	});
	it('rejects invalid values', () => {
		expect(parseDollars('abc')).toEqual({ ok: false, reason: 'invalid' });
		expect(parseDollars('1.2.3')).toEqual({ ok: false, reason: 'invalid' });
		expect(parseDollars('-1')).toEqual({ ok: false, reason: 'negative' });
		expect(parseDollars('1.999')).toEqual({ ok: false, reason: 'precision' });
		expect(parseDollars('1e5')).toEqual({ ok: false, reason: 'invalid' });
		expect(parseDollars('99999999')).toEqual({ ok: false, reason: 'too_large' });
	});
});

describe('centsToDollarInput', () => {
	it('formats cents for inputs', () => {
		expect(centsToDollarInput(1999)).toBe('19.99');
		expect(centsToDollarInput(5)).toBe('0.05');
		expect(centsToDollarInput(100)).toBe('1.00');
		expect(centsToDollarInput(null)).toBe('');
	});
	it('round-trips with parseDollars', () => {
		for (const c of [0, 1, 99, 100, 12345, 99999999]) {
			expect(parseDollars(centsToDollarInput(c))).toEqual({ ok: true, cents: c });
		}
	});
});

describe('formatCAD', () => {
	it('formats with en-CA currency style', () => {
		expect(formatCAD(1999)).toBe('$19.99');
		expect(formatCAD(123456)).toBe('$1,234.56');
	});
});
