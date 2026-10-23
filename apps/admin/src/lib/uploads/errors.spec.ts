import { describe, expect, it } from 'vitest';
import { m } from '$lib/paraglide/messages.js';
import { uploadErrorCodeForStatus, uploadErrorMessage } from './errors';

describe('upload errors', () => {
	it('gives damaged files their own message', () => {
		expect(uploadErrorMessage('unreadable')).toBe(m.up_err_unreadable());
		expect(uploadErrorMessage('unreadable')).not.toBe(uploadErrorMessage('unsupported'));
	});
	it('falls back to a generic message', () => {
		expect(uploadErrorMessage(undefined)).toBe(m.up_err_failed());
	});
	it('maps rejected request statuses', () => {
		expect(uploadErrorCodeForStatus(413)).toBe('too_large');
		expect(uploadErrorCodeForStatus(415)).toBe('unsupported');
		expect(uploadErrorCodeForStatus(422)).toBeUndefined();
	});
});
