import type { UploadJob } from '@atcreations/api-types';
import { m } from '$lib/paraglide/messages.js';

export type UploadErrorCode = NonNullable<UploadJob['errorCode']>;

/** Mirrors the API's upload limit (MaxUploadBytes). */
export const MAX_UPLOAD_MB = 25;

const messages: Record<UploadErrorCode, () => string> = {
	unsupported: m.up_err_unsupported,
	unreadable: m.up_err_unreadable,
	too_large: () => m.up_err_too_large({ max: MAX_UPLOAD_MB }),
	dimensions: m.up_err_dimensions,
	limit: m.up_err_limit,
	unavailable: m.up_err_unavailable,
	failed: m.up_err_failed
};

/** Localized reason for a failed upload (the API's codes are stable; its text is English). */
export function uploadErrorMessage(code: UploadErrorCode | undefined): string {
	return (code && messages[code]?.()) || m.up_err_failed();
}

/** Error code for a rejected upload request, when its HTTP status identifies one. */
export function uploadErrorCodeForStatus(status: number | undefined): UploadErrorCode | undefined {
	if (status === 413) return 'too_large';
	if (status === 415) return 'unsupported';
	return undefined;
}
