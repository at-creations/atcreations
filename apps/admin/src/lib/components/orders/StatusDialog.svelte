<script lang="ts">
	import type { Order, OrderStatus } from '@atcreations/api-types';
	import { fieldErrorMap } from '$lib/api';
	import Field from '$lib/components/Field.svelte';
	import Icon from '$lib/components/Icon.svelte';
	import Modal from '$lib/components/Modal.svelte';
	import { orderStatusLabel } from '$lib/i18n';
	import { toastError } from '$lib/notify';
	import { m } from '$lib/paraglide/messages.js';
	import { createOrderStatusMutation } from '$lib/queries';
	import { toast } from '$lib/state/toast.svelte';

	let {
		open = $bindable(false),
		order,
		target
	}: {
		open?: boolean;
		order: Order;
		target: OrderStatus | null;
	} = $props();

	const uid = $props.id();
	let note = $state('');
	let trackingNumber = $state('');
	let notify = $state(true);
	const changeStatus = createOrderStatusMutation();
	const saving = $derived(changeStatus.isPending);
	let errors = $state<Record<string, string>>({});

	export function reset() {
		note = '';
		trackingNumber = order.trackingNumber ?? '';
		notify = true;
		errors = {};
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (!target || saving) return;
		const e: Record<string, string> = {};
		if (note.length > 1000) e.note = m.validation_too_long({ max: 1000 });
		if (trackingNumber.length > 100) e.trackingNumber = m.validation_too_long({ max: 100 });
		errors = e;
		if (Object.keys(e).length) return;
		try {
			await changeStatus.mutateAsync({
				id: order.id,
				status: target,
				...(note.trim() ? { note: note.trim() } : {}),
				...(target === 'shipped' && trackingNumber.trim()
					? { trackingNumber: trackingNumber.trim() }
					: {}),
				notify
			});
			toast.success(m.od_status_updated());
			open = false;
		} catch (error) {
			errors = fieldErrorMap(error);
			toastError(error, m.od_status_failed());
		}
	}
</script>

<Modal
	bind:open
	busy={saving}
	title={target ? m.od_transition_title({ status: orderStatusLabel(target) }) : ''}
	description={target
		? m.od_transition_from_to({
				from: orderStatusLabel(order.status),
				to: orderStatusLabel(target)
			})
		: undefined}
>
	<form id="{uid}-form" class="space-y-4" onsubmit={submit} novalidate>
		{#if target === 'cancelled'}
			<p class="flex items-start gap-2 rounded-lg bg-amber-50 p-3 text-sm text-amber-900">
				<Icon name="alert" class="mt-0.5" />{m.od_cancel_warning()}
			</p>
		{/if}
		{#if target === 'shipped'}
			<Field label={m.od_tracking()} optional error={errors.trackingNumber}>
				{#snippet children(control)}
					<input {...control} class="input font-mono" maxlength="100" bind:value={trackingNumber} />
				{/snippet}
			</Field>
		{/if}
		<Field label={m.od_transition_note()} error={errors.note}>
			{#snippet children(control)}
				<textarea {...control} class="input min-h-20" maxlength="1000" bind:value={note}></textarea>
			{/snippet}
		</Field>
		<label class="inline-flex items-center gap-2 text-sm">
			<input type="checkbox" class="size-4 rounded border-zinc-300" bind:checked={notify} />
			{m.od_notify()}
		</label>
	</form>
	{#snippet footer()}
		<button type="button" class="btn-secondary" disabled={saving} onclick={() => (open = false)}>
			{m.common_cancel()}
		</button>
		<button
			type="submit"
			form="{uid}-form"
			class={target === 'cancelled' || target === 'refunded' ? 'btn-danger' : 'btn-primary'}
			disabled={saving}
		>
			{saving ? m.common_processing() : m.common_confirm()}
		</button>
	{/snippet}
</Modal>
