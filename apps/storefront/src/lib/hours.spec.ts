import { describe, expect, it } from 'vitest';
import type { BusinessHours } from '@atcreations/api-types';
import { groupBusinessHours, todaysHours } from './hours';

const open = (day: BusinessHours['day'], o = '10:00', c = '18:00'): BusinessHours => ({
	day,
	closed: false,
	open: o,
	close: c
});
const closed = (day: BusinessHours['day']): BusinessHours => ({ day, closed: true });

describe('groupBusinessHours', () => {
	it('returns nothing for empty input', () => {
		expect(groupBusinessHours([])).toEqual([]);
	});

	it('groups consecutive identical days', () => {
		const hours = [
			open('mon'),
			open('tue'),
			open('wed'),
			open('thu'),
			open('fri'),
			open('sat', '11:00', '17:00'),
			closed('sun')
		];
		expect(groupBusinessHours(hours)).toEqual([
			{ from: 'mon', to: 'fri', closed: false, open: '10:00', close: '18:00' },
			{ from: 'sat', to: 'sat', closed: false, open: '11:00', close: '17:00' },
			{ from: 'sun', to: 'sun', closed: true }
		]);
	});

	it('sorts by weekday and does not merge non-consecutive days', () => {
		const hours = [open('fri'), closed('tue'), open('mon'), open('wed')];
		expect(groupBusinessHours(hours)).toEqual([
			{ from: 'mon', to: 'mon', closed: false, open: '10:00', close: '18:00' },
			{ from: 'tue', to: 'tue', closed: true },
			{ from: 'wed', to: 'wed', closed: false, open: '10:00', close: '18:00' },
			{ from: 'fri', to: 'fri', closed: false, open: '10:00', close: '18:00' }
		]);
	});

	it('treats missing times as closed and merges closed runs', () => {
		const hours: BusinessHours[] = [{ day: 'sat', closed: false }, closed('sun')];
		expect(groupBusinessHours(hours)).toEqual([{ from: 'sat', to: 'sun', closed: true }]);
	});
});

describe('todaysHours', () => {
	const hours = [
		{ day: 'mon' as const, open: '11:00', close: '19:00', closed: false },
		{ day: 'tue' as const, closed: true }
	];
	it("uses the shop's time zone", () => {
		// 2026-10-27 03:00 UTC is still Monday evening in Vancouver.
		expect(todaysHours(hours, new Date('2026-10-27T03:00:00Z'))).toEqual({
			open: '11:00',
			close: '19:00'
		});
		expect(todaysHours(hours, new Date('2026-10-27T20:00:00Z'))).toBe('closed');
		expect(todaysHours(hours, new Date('2026-10-28T20:00:00Z'))).toBeNull();
	});
});
