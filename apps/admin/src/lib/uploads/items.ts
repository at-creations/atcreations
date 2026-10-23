import type { UploadJob } from '@atcreations/api-types';
import { m } from '$lib/paraglide/messages.js';
import type { LocalUpload } from './uploader.svelte';

/** One tile in the upload strip: a file still being sent, or a job on the server. */
export interface UploadItem {
	key: string;
	name: string;
	preview?: string;
	phase: 'sending' | 'waiting' | 'queued' | 'processing' | 'retrying' | 'failed';
	/** 0-100 for the current phase. */
	percent: number;
	label: string;
	error?: string;
	/** Local Uppy file id (sending/failed before reaching the server). */
	fileId?: string;
	/** Server job id. */
	jobId?: string;
	/** Queued and failed jobs can be cancelled/dismissed; failed local files can be retried. */
	canCancel: boolean;
	canRetry: boolean;
}

const stageLabels: Record<string, () => string> = {
	waiting: m.up_stage_waiting,
	downloading: m.up_stage_downloading,
	decoding: m.up_stage_decoding,
	resizing: m.up_stage_resizing,
	encoding: m.up_stage_encoding,
	uploading: m.up_stage_saving
};

export function uploadItems(local: LocalUpload[], jobs: UploadJob[] | undefined): UploadItem[] {
	const items: UploadItem[] = [];
	for (const f of local) {
		if (f.jobId) continue; // the server job below represents it now
		items.push({
			key: `f:${f.id}`,
			name: f.name,
			preview: f.preview,
			fileId: f.id,
			phase: f.state === 'error' ? 'failed' : f.state === 'waiting' ? 'waiting' : 'sending',
			percent: f.progress,
			label:
				f.state === 'error'
					? m.up_failed()
					: f.state === 'waiting'
						? m.up_waiting_retry()
						: m.up_sending({ percent: f.progress }),
			error: f.error,
			canCancel: f.state === 'error',
			canRetry: f.state === 'error'
		});
	}
	for (const j of jobs ?? []) {
		if (j.status === 'done' || j.status === 'cancelled') continue;
		const retrying = j.status === 'queued' && j.error === 'retrying';
		const preview = local.find((f) => f.jobId === j.id)?.preview;
		items.push({
			key: `j:${j.id}`,
			name: j.fileName,
			preview,
			jobId: j.id,
			phase:
				j.status === 'failed'
					? 'failed'
					: retrying
						? 'retrying'
						: j.status === 'queued'
							? 'queued'
							: 'processing',
			percent: j.status === 'processing' ? j.percent : 0,
			label:
				j.status === 'failed'
					? m.up_failed()
					: retrying
						? m.up_retrying()
						: j.status === 'queued'
							? m.up_queued()
							: `${stageLabels[j.stage ?? 'waiting']?.() ?? m.up_processing()} · ${j.percent}%`,
			error: j.status === 'failed' ? j.error : undefined,
			canCancel: j.status === 'queued' || j.status === 'failed',
			canRetry: false
		});
	}
	return items;
}
