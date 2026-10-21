import { describe, expect, it } from 'vitest';
import { ApiError, fieldErrorMap, parseProblem } from './api-error';

describe('parseProblem', () => {
	it('parses problem+json bodies', () => {
		const e = parseProblem(422, {
			type: 'about:blank',
			title: 'Validation failed',
			status: 422,
			detail: 'Some fields are invalid',
			requestId: 'req-1',
			errors: [{ field: 'name.en', message: 'required' }, { bogus: true }]
		});
		expect(e).toBeInstanceOf(ApiError);
		expect(e.status).toBe(422);
		expect(e.title).toBe('Validation failed');
		expect(e.detail).toBe('Some fields are invalid');
		expect(e.requestId).toBe('req-1');
		expect(e.errors).toEqual([{ field: 'name.en', message: 'required' }]);
	});
	it('falls back for non-JSON bodies', () => {
		const e = parseProblem(502, 'Bad gateway');
		expect(e.status).toBe(502);
		expect(e.title).toBe('HTTP 502');
		expect(e.detail).toBe('Bad gateway');
	});
});

describe('fieldErrorMap', () => {
	it('maps the first error per field and strips prefixes', () => {
		const e = parseProblem(422, {
			title: 'x',
			status: 422,
			errors: [
				{ field: 'shipping.city', message: 'required' },
				{ field: 'shipping.city', message: 'too long' },
				{ field: 'slug', message: 'taken' }
			]
		});
		expect(fieldErrorMap(e, 'shipping.')).toEqual({ city: 'required', slug: 'taken' });
		expect(fieldErrorMap(new Error('x'))).toEqual({});
	});
});
