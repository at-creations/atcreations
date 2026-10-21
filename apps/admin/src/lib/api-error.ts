export interface FieldError {
	field: string;
	message: string;
}

/** Typed view of an RFC 9457 problem+json response. `status: 0` means a network failure. */
export class ApiError extends Error {
	readonly status: number;
	readonly title: string;
	readonly detail?: string;
	readonly errors: FieldError[];
	readonly requestId?: string;

	constructor(init: {
		status: number;
		title: string;
		detail?: string;
		errors?: FieldError[];
		requestId?: string;
	}) {
		super(init.detail || init.title);
		this.name = 'ApiError';
		this.status = init.status;
		this.title = init.title;
		this.detail = init.detail;
		this.errors = init.errors ?? [];
		this.requestId = init.requestId;
	}
}

function str(value: unknown): string | undefined {
	return typeof value === 'string' && value.length > 0 ? value : undefined;
}

/** Build an ApiError from an HTTP status and whatever body the server returned. */
export function parseProblem(status: number, body: unknown): ApiError {
	if (body && typeof body === 'object') {
		const b = body as Record<string, unknown>;
		const errors = Array.isArray(b.errors)
			? b.errors
					.filter(
						(e): e is FieldError =>
							!!e &&
							typeof e === 'object' &&
							typeof (e as FieldError).field === 'string' &&
							typeof (e as FieldError).message === 'string'
					)
					.map((e) => ({ field: e.field, message: e.message }))
			: [];
		return new ApiError({
			status: typeof b.status === 'number' ? b.status : status,
			title: str(b.title) ?? `HTTP ${status}`,
			detail: str(b.detail),
			errors,
			requestId: str(b.requestId)
		});
	}
	return new ApiError({ status, title: `HTTP ${status}`, detail: str(body), errors: [] });
}

/**
 * Map API field errors onto form keys. Field paths like "name.en" are kept verbatim; a `prefix`
 * (e.g. "shipping.") is stripped so nested forms can reuse the same keys.
 */
export function fieldErrorMap(error: unknown, prefix = ''): Record<string, string> {
	if (!(error instanceof ApiError)) return {};
	const out: Record<string, string> = {};
	for (const e of error.errors) {
		const key = prefix && e.field.startsWith(prefix) ? e.field.slice(prefix.length) : e.field;
		if (!(key in out)) out[key] = e.message;
	}
	return out;
}
