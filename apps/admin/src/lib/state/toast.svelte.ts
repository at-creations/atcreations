export type ToastKind = 'success' | 'error' | 'info';

export interface Toast {
	id: number;
	kind: ToastKind;
	message: string;
	detail?: string;
}

let nextId = 1;
// Plain record: timers are bookkeeping, not reactive UI state.
const timers: Record<number, ReturnType<typeof setTimeout>> = {};

/** Global toast queue. Rendered by the single <Toaster /> in the root layout. */
export const toasts = $state<Toast[]>([]);

export function dismissToast(id: number) {
	const index = toasts.findIndex((t) => t.id === id);
	if (index !== -1) toasts.splice(index, 1);
	const timer = timers[id];
	if (timer) clearTimeout(timer);
	delete timers[id];
}

function push(kind: ToastKind, message: string, detail?: string, timeout?: number) {
	const id = nextId++;
	toasts.push({ id, kind, message, detail });
	if (toasts.length > 5) dismissToast(toasts[0]!.id);
	const ms = timeout ?? (kind === 'error' ? 8000 : 4000);
	if (ms > 0) timers[id] = setTimeout(() => dismissToast(id), ms);
	return id;
}

export const toast = {
	success: (message: string, detail?: string) => push('success', message, detail),
	error: (message: string, detail?: string) => push('error', message, detail),
	info: (message: string, detail?: string) => push('info', message, detail)
};
