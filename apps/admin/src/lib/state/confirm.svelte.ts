export interface ConfirmOptions {
	title: string;
	message: string;
	/** Extra warning line (shown emphasised). */
	warning?: string;
	confirmLabel?: string;
	cancelLabel?: string;
	danger?: boolean;
	/** When set, the user must type this exact text before confirming. */
	requireText?: string;
	requireTextLabel?: string;
}

interface PendingConfirm extends ConfirmOptions {
	resolve: (value: boolean) => void;
}

/** The single confirmation request currently shown by <ConfirmHost />. */
export const confirmState = $state<{ current: PendingConfirm | null }>({ current: null });

/** Ask the user to confirm an action. Resolves `false` when cancelled or dismissed. */
export function confirmAction(options: ConfirmOptions): Promise<boolean> {
	confirmState.current?.resolve(false);
	return new Promise<boolean>((resolve) => {
		confirmState.current = { ...options, resolve };
	});
}

export function settleConfirm(value: boolean) {
	const current = confirmState.current;
	confirmState.current = null;
	current?.resolve(value);
}
