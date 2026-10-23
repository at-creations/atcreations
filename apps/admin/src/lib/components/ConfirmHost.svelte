<script lang="ts">
	import { m } from '$lib/paraglide/messages.js';
	import { confirmState, settleConfirm } from '$lib/state/confirm.svelte';
	import Icon from './Icon.svelte';
	import Modal from './Modal.svelte';

	const request = $derived(confirmState.current);
	let typed = $state('');
	const uid = $props.id();

	const canConfirm = $derived(!request?.requireText || typed.trim() === request.requireText);

	function finish(value: boolean) {
		typed = '';
		settleConfirm(value);
	}
</script>

<Modal open={!!request} title={request?.title ?? ''} size="sm" onclose={() => finish(false)}>
	{#if request}
		<form
			id="{uid}-form"
			onsubmit={(event) => {
				event.preventDefault();
				if (canConfirm) finish(true);
			}}
		>
			<p class="text-sm break-words whitespace-pre-line text-zinc-700">{request.message}</p>
			{#if request.warning}
				<p
					class="mt-3 flex items-start gap-2 rounded-lg bg-amber-50 p-3 text-sm font-medium text-amber-900"
				>
					<Icon name="alert" class="mt-0.5" />
					<span>{request.warning}</span>
				</p>
			{/if}
			{#if request.requireText}
				<label class="mt-4 label" for="{uid}-typed">{request.requireTextLabel}</label>
				<input
					id="{uid}-typed"
					class="input"
					autocomplete="off"
					spellcheck="false"
					bind:value={typed}
				/>
			{/if}
		</form>
	{/if}
	{#snippet footer()}
		<button type="button" class="btn-secondary" onclick={() => finish(false)}>
			{request?.cancelLabel ?? m.common_cancel()}
		</button>
		<button
			type="submit"
			form="{uid}-form"
			class={request?.danger ? 'btn-danger' : 'btn-primary'}
			disabled={!canConfirm}
		>
			{request?.confirmLabel ?? m.common_confirm()}
		</button>
	{/snippet}
</Modal>
