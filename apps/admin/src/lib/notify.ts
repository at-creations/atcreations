import { describeError } from '$lib/i18n';
import { toast } from '$lib/state/toast.svelte';

/** Show a failed request as an error toast (problem `detail` is surfaced as plain text). */
export function toastError(error: unknown, fallback?: string) {
	const { message, detail } = describeError(error, fallback);
	toast.error(message, detail);
}
