import { describe, expect, it } from 'vitest';
import { ApiError } from '$lib/api-error';
import { createAdminQueryClient, GC_TIME, MAX_RETRIES, shouldRetry, STALE_TIME } from './client';

const apiError = (status: number) => new ApiError({ status, title: `HTTP ${status}` });

describe('shouldRetry', () => {
	it.each([400, 401, 403, 404, 409, 422, 429])('never retries a %i', (status) => {
		expect(shouldRetry(0, apiError(status))).toBe(false);
	});

	it.each([0, 500, 502, 503])('retries status %i up to the limit', (status) => {
		expect(shouldRetry(0, apiError(status))).toBe(true);
		expect(shouldRetry(MAX_RETRIES - 1, apiError(status))).toBe(true);
		expect(shouldRetry(MAX_RETRIES, apiError(status))).toBe(false);
	});

	it('allows at most two retries', () => {
		expect(MAX_RETRIES).toBe(2);
	});

	it('does not retry unknown errors', () => {
		expect(shouldRetry(0, new TypeError('boom'))).toBe(false);
		expect(shouldRetry(0, 'nope')).toBe(false);
	});
});

describe('createAdminQueryClient', () => {
	it('applies the admin defaults', () => {
		const client = createAdminQueryClient();
		const defaults = client.getDefaultOptions().queries;
		expect(defaults?.staleTime).toBe(STALE_TIME);
		expect(STALE_TIME).toBe(30_000);
		expect(defaults?.gcTime).toBe(GC_TIME);
		expect(GC_TIME).toBe(300_000);
		expect(defaults?.refetchOnWindowFocus).toBe(true);
		expect(defaults?.retry).toBe(shouldRetry);
	});

	it('creates independent caches', () => {
		const a = createAdminQueryClient();
		const b = createAdminQueryClient();
		a.setQueryData(['me'], { id: 'u1' });
		expect(b.getQueryData(['me'])).toBeUndefined();
		a.clear();
		expect(a.getQueryData(['me'])).toBeUndefined();
	});
});
