import type { BusinessHours } from '@atcreations/api-types';

export type DayKey = BusinessHours['day'];
export const DAY_ORDER: readonly DayKey[] = ['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun'];

export interface HoursGroup {
	from: DayKey;
	to: DayKey;
	closed: boolean;
	open?: string;
	close?: string;
}

function signature(h: BusinessHours): string {
	return h.closed || !h.open || !h.close ? 'closed' : `${h.open}-${h.close}`;
}

/**
 * Sort business hours by weekday and merge *consecutive* days with identical hours
 * (e.g. Monday – Friday 10:00–18:00). Entries missing open/close are treated as closed.
 */
export function groupBusinessHours(hours: readonly BusinessHours[]): HoursGroup[] {
	const byDay = new Map<DayKey, BusinessHours>();
	for (const h of hours) if (DAY_ORDER.includes(h.day)) byDay.set(h.day, h);

	const groups: HoursGroup[] = [];
	let prevIndex = -2;
	let prevSig = '';
	for (const [index, day] of DAY_ORDER.entries()) {
		const h = byDay.get(day);
		if (!h) continue;
		const sig = signature(h);
		const last = groups.at(-1);
		if (last && prevIndex === index - 1 && prevSig === sig) {
			last.to = day;
		} else {
			groups.push(
				sig === 'closed'
					? { from: day, to: day, closed: true }
					: { from: day, to: day, closed: false, open: h.open, close: h.close }
			);
		}
		prevIndex = index;
		prevSig = sig;
	}
	return groups;
}
