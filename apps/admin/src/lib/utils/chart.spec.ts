import { describe, expect, it } from 'vitest';
import { change, compactNumber, niceTicks } from './chart';

describe('niceTicks', () => {
	it('uses 1/2/5 steps that cover the maximum', () => {
		expect(niceTicks(9)).toEqual([0, 5, 10]);
		expect(niceTicks(37)).toEqual([0, 10, 20, 30, 40]);
		expect(niceTicks(1250)).toEqual([0, 500, 1000, 1500]);
		expect(niceTicks(0.8)).toEqual([0, 0.2, 0.4, 0.6, 0.8]);
	});
	it('gives an empty series a 0–1 axis', () => {
		expect(niceTicks(0)).toEqual([0, 1]);
		expect(niceTicks(Number.NaN)).toEqual([0, 1]);
	});
});

describe('compactNumber', () => {
	it('abbreviates large numbers', () => {
		expect(compactNumber(950)).toBe('950');
		expect(compactNumber(1284)).toBe('1.3K');
		expect(compactNumber(4_200_000)).toBe('4.2M');
	});
});

describe('change', () => {
	it('returns the relative change, or null without a baseline', () => {
		expect(change(150, 100)).toBe(0.5);
		expect(change(50, 100)).toBe(-0.5);
		expect(change(10, 0)).toBeNull();
	});
});
