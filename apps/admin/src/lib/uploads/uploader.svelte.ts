import type { UploadJob } from '@atcreations/api-types';
import Uppy from '@uppy/core';
import XHRUpload from '@uppy/xhr-upload';
import { apiBaseUrl, parseProblem, reportUnauthorized, sessionToken } from '$lib/api';
import { MAX_UPLOAD_MB, uploadErrorCodeForStatus, uploadErrorMessage } from './errors';

export type UploadKind = 'product' | 'category';

/** Accepted formats; HEIC/HEIF (iPhone) is converted by the image worker. */
export const UPLOAD_TYPES = [
	'image/jpeg',
	'image/png',
	'image/webp',
	'image/heic',
	'image/heif',
	'.jpg',
	'.jpeg',
	'.png',
	'.webp',
	'.heic',
	'.heif'
];
export { MAX_UPLOAD_MB };
/** Files sent at once. The API buffers a few per instance; more would only queue there. */
const PARALLEL = 3;

type Meta = Record<string, never>;
type Body = { data?: UploadJob };

export interface LocalUpload {
	/** Uppy file id. */
	id: string;
	name: string;
	size: number;
	/** Object URL for a preview (absent for HEIC, which browsers cannot display). */
	preview?: string;
	/** Bytes sent, 0-100. */
	progress: number;
	state: 'uploading' | 'waiting' | 'error';
	error?: string;
	/** Set once the API accepted the file and queued a processing job. */
	jobId?: string;
}

const RETRYABLE = new Set([0, 408, 429, 500, 502, 503, 504]);

/**
 * Wraps Uppy for one product or category: parallel uploads with progress, retries that respect
 * the API's Retry-After (shared by every file, since rate limits are per client), a fresh session
 * token per attempt, and reactive per-file state for the UI.
 */
export class Uploader {
	files = $state<LocalUpload[]>([]);
	#uppy: Uppy<Meta, Body>;
	#pauseUntil = 0;
	#onQueued: (job: UploadJob, fileId: string) => void;

	constructor(
		kind: UploadKind,
		ownerId: string,
		onQueued: (job: UploadJob, fileId: string) => void
	) {
		this.#onQueued = onQueued;
		const endpoint = `${apiBaseUrl()}/v1/admin/${kind === 'product' ? 'products' : 'categories'}/${encodeURIComponent(ownerId)}/uploads`;
		this.#uppy = new Uppy<Meta, Body>({
			autoProceed: true,
			allowMultipleUploadBatches: true,
			restrictions: { maxFileSize: MAX_UPLOAD_MB * 1024 * 1024, allowedFileTypes: UPLOAD_TYPES }
		}).use(XHRUpload<Meta, Body>, {
			endpoint,
			fieldName: 'file',
			formData: true,
			limit: PARALLEL,
			timeout: 120_000,
			// Uppy retries each file 3 times with a short back-off, only for network errors, 408,
			// 429 and 5xx; on top of that every retry waits out the API's Retry-After. A file that
			// still fails can be retried from its tile.
			shouldRetry: (xhr) => RETRYABLE.has(xhr.status),
			onBeforeRequest: async (xhr, retryCount) => {
				const wait = this.#pauseUntil - Date.now();
				if (retryCount > 0 && wait > 0) await sleep(wait + Math.random() * 500);
				const token = await sessionToken();
				if (token) xhr.setRequestHeader('Authorization', `Bearer ${token}`);
			},
			onAfterResponse: (xhr) => {
				if (xhr.status === 401) reportUnauthorized();
				if (xhr.status === 429 || xhr.status === 503) {
					const seconds = Number(xhr.getResponseHeader('Retry-After'));
					const ms = Number.isFinite(seconds) && seconds > 0 ? Math.min(seconds, 120) * 1000 : 5000;
					this.#pauseUntil = Math.max(this.#pauseUntil, Date.now() + ms);
				}
			},
			getResponseData: (xhr) => {
				try {
					return JSON.parse(xhr.responseText) as Body;
				} catch {
					return {};
				}
			}
		});

		this.#uppy.on('file-added', (file) => {
			const previewable =
				typeof file.type === 'string' && /^image\/(jpeg|png|webp)$/.test(file.type);
			this.files.push({
				id: file.id,
				name: file.name ?? 'image',
				size: file.size ?? 0,
				preview:
					previewable && file.data instanceof Blob ? URL.createObjectURL(file.data) : undefined,
				progress: 0,
				state: 'uploading'
			});
		});
		this.#uppy.on('upload-progress', (file, p) => {
			const f = file && this.#find(file.id);
			if (f && p.bytesTotal) {
				f.progress = Math.round(((p.bytesUploaded ?? 0) / p.bytesTotal) * 100);
				f.state = 'uploading';
			}
		});
		this.#uppy.on('upload-retry', (file) => {
			const f = file && this.#find(file.id);
			if (f) {
				f.state = 'waiting';
				f.progress = 0;
			}
		});
		this.#uppy.on('upload-success', (file, response) => {
			const f = file && this.#find(file.id);
			const job = response.body?.data;
			if (f) {
				f.progress = 100;
				f.jobId = job?.id;
			}
			if (job && file) this.#onQueued(job, file.id);
		});
		this.#uppy.on('upload-error', (file, error, response) => {
			const f = file && this.#find(file.id);
			if (!f) return;
			f.state = 'error';
			const code = uploadErrorCodeForStatus(response?.status);
			f.error = code
				? uploadErrorMessage(code)
				: (problemMessage(response?.body, response?.status) ?? error.message);
		});
		this.#uppy.on('restriction-failed', (file) => {
			// Uppy ids derive from the file itself: picking the same rejected file twice (or a
			// duplicate of one already listed) must not add a second tile with the same id.
			if (file && !this.#find(file.id))
				this.files.push({
					id: file.id,
					name: file.name ?? 'image',
					size: file.size ?? 0,
					progress: 0,
					state: 'error',
					// Only type and size are restricted; anything within the size limit has the wrong type.
					error: uploadErrorMessage(
						(file.size ?? 0) > MAX_UPLOAD_MB * 1024 * 1024 ? 'too_large' : 'unsupported'
					)
				});
		});
	}

	#find(id: string) {
		return this.files.find((f) => f.id === id);
	}

	/** Queue files for upload; returns how many were accepted by the restrictions. */
	add(files: FileList | File[] | null | undefined, room = Infinity): number {
		let added = 0;
		for (const data of Array.from(files ?? [])) {
			if (added >= room) break;
			try {
				this.#uppy.addFile({ name: data.name, type: data.type, data, source: 'local' });
				added++;
			} catch {
				// Restriction failures are reported through 'restriction-failed'.
			}
		}
		return added;
	}

	retry(id: string) {
		// Files rejected by the restrictions never entered Uppy: nothing to retry.
		if (!this.#uppy.getFile(id)) {
			this.dismiss(id);
			return;
		}
		const f = this.#find(id);
		if (f) {
			f.state = 'uploading';
			f.error = undefined;
		}
		this.#uppy.retryUpload(id).catch(() => {
			// Failures are reported through 'upload-error'.
		});
	}

	/** Whether a local file can be retried (it reached Uppy, i.e. was not rejected up front). */
	canRetry(id: string) {
		return !!this.#uppy.getFile(id);
	}

	/** Forget a local upload (after a failure, or once its server job is visible). */
	dismiss(id: string) {
		const f = this.#find(id);
		if (f?.preview) URL.revokeObjectURL(f.preview);
		this.files = this.files.filter((x) => x.id !== id);
		if (this.#uppy.getFile(id)) this.#uppy.removeFile(id);
	}

	/** Uploads not yet handed over to a server job (still sending, waiting or failed). */
	get pending() {
		return this.files.filter((f) => !f.jobId);
	}

	get busy() {
		return this.files.some((f) => !f.jobId && f.state !== 'error');
	}

	destroy() {
		for (const f of this.files) if (f.preview) URL.revokeObjectURL(f.preview);
		this.#uppy.destroy();
	}
}

function sleep(ms: number) {
	return new Promise((resolve) => setTimeout(resolve, ms));
}

function problemMessage(body: unknown, status?: number): string | undefined {
	if (!body || typeof body !== 'object') return undefined;
	try {
		return parseProblem(status ?? 0, body).detail || undefined;
	} catch {
		return undefined;
	}
}
