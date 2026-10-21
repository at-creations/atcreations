import { describe, expect, it } from 'vitest';
import { normalizeHours, toHoursInput, validateHours } from './hours';

describe('business hours', () => {
	it('normalizes to seven ordered days', () => {
		const hours = normalizeHours([{ day: 'sun', closed: true }]);
		expect(hours.map((h) => h.day)).toEqual(['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun']);
		expect(hours[6]?.closed).toBe(true);
	});
	it('validates open < close', () => {
		expect(validateHours({ day: 'mon', closed: false, open: '09:00', close: '17:00' })).toBeNull();
		expect(validateHours({ day: 'mon', closed: false, open: '17:00', close: '09:00' })).toBe(
			'order'
		);
		expect(validateHours({ day: 'mon', closed: false, open: '09:00', close: '09:00' })).toBe(
			'order'
		);
		expect(validateHours({ day: 'mon', closed: false, open: '', close: '09:00' })).toBe('required');
		expect(validateHours({ day: 'mon', closed: true })).toBeNull();
	});
	it('drops times for closed days', () => {
		expect(toHoursInput([{ day: 'mon', closed: true, open: '09:00', close: '10:00' }])).toEqual([
			{ day: 'mon', closed: true }
		]);
	});
});
