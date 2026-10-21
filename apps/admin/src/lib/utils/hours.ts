import type { BusinessHours } from '@atcreations/api-types';

export type Day = BusinessHours['day'];
export const DAYS: readonly Day[] = ['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun'];
const TIME = /^([01]\d|2[0-3]):[0-5]\d$/;

/** Always return exactly one entry per weekday, in Monday→Sunday order. */
export function normalizeHours(hours: readonly BusinessHours[] | undefined): BusinessHours[] {
	return DAYS.map((day) => {
		const found = hours?.find((h) => h.day === day);
		if (found)
			return { day, closed: found.closed, open: found.open ?? '', close: found.close ?? '' };
		return { day, closed: false, open: '10:00', close: '18:00' };
	});
}

export type HoursError = 'required' | 'order';

/** Validate one day: open days need HH:MM times with open < close. */
export function validateHours(h: BusinessHours): HoursError | null {
	if (h.closed) return null;
	if (!h.open || !h.close || !TIME.test(h.open) || !TIME.test(h.close)) return 'required';
	return h.open < h.close ? null : 'order';
}

/** Payload for the API: closed days carry no times. */
export function toHoursInput(hours: readonly BusinessHours[]): BusinessHours[] {
	return hours.map((h) =>
		h.closed
			? { day: h.day, closed: true }
			: { day: h.day, closed: false, open: h.open, close: h.close }
	);
}
