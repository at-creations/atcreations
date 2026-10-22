<script lang="ts">
	import type { Address, Order } from '@atcreations/api-types';
	import type { FullAutoFill } from 'svelte/elements';
	import { fieldErrorMap } from '$lib/api';
	import Field from '$lib/components/Field.svelte';
	import Modal from '$lib/components/Modal.svelte';
	import { toastError } from '$lib/notify';
	import { m } from '$lib/paraglide/messages.js';
	import { createOrderShippingMutation } from '$lib/queries';
	import { toast } from '$lib/state/toast.svelte';

	let { open = $bindable(false), order }: { open?: boolean; order: Order } = $props();

	const uid = $props.id();
	type Key = keyof Address;
	const empty: Address = {
		name: '',
		phone: '',
		line1: '',
		line2: '',
		city: '',
		region: '',
		postalCode: '',
		country: 'CA'
	};
	let form = $state<Address>({ ...empty });
	let errors = $state<Record<string, string>>({});
	const saveShipping = createOrderShippingMutation();
	const saving = $derived(saveShipping.isPending);

	export function reset() {
		form = { ...empty, ...order.shipping };
		errors = {};
	}

	const fields: {
		key: Key;
		label: () => string;
		max: number;
		required: boolean;
		hint?: () => string;
		auto: FullAutoFill;
	}[] = [
		{ key: 'name', label: m.addr_name, max: 100, required: true, auto: 'name' },
		{ key: 'phone', label: m.addr_phone, max: 40, required: true, auto: 'tel' },
		{ key: 'line1', label: m.addr_line1, max: 200, required: true, auto: 'address-line1' },
		{ key: 'line2', label: m.addr_line2, max: 200, required: false, auto: 'address-line2' },
		{ key: 'city', label: m.addr_city, max: 100, required: true, auto: 'address-level2' },
		{ key: 'region', label: m.addr_region, max: 100, required: false, auto: 'address-level1' },
		{ key: 'postalCode', label: m.addr_postal, max: 20, required: true, auto: 'postal-code' },
		{
			key: 'country',
			label: m.addr_country,
			max: 2,
			required: true,
			hint: m.addr_country_hint,
			auto: 'country'
		}
	];

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (saving) return;
		const e: Record<string, string> = {};
		const body: Address = { ...empty };
		for (const f of fields) {
			const value = (form[f.key] ?? '').trim();
			if (f.required && !value) e[f.key] = m.validation_required();
			else if (value.length > f.max) e[f.key] = m.validation_too_long({ max: f.max });
			body[f.key] = value;
		}
		body.country = body.country.toUpperCase();
		if (!e.country && !/^[A-Z]{2}$/.test(body.country)) e.country = m.addr_country_invalid();
		errors = e;
		if (Object.keys(e).length) return;
		if (!body.line2) delete body.line2;
		if (!body.region) delete body.region;
		try {
			await saveShipping.mutateAsync({ id: order.id, address: body });
			toast.success(m.od_address_saved());
			open = false;
		} catch (error) {
			errors = fieldErrorMap(error);
			toastError(error, m.od_address_failed());
		}
	}
</script>

<Modal bind:open busy={saving} size="lg" title={m.od_edit_address()}>
	<form id="{uid}-form" class="grid gap-4 sm:grid-cols-2" onsubmit={submit} novalidate>
		{#each fields as f (f.key)}
			<Field
				label={f.label()}
				required={f.required}
				hint={f.hint?.()}
				error={errors[f.key]}
				class={f.key === 'line1' || f.key === 'line2' ? 'sm:col-span-2' : ''}
			>
				{#snippet children(control)}
					<input
						{...control}
						class="input"
						maxlength={f.max}
						autocomplete={f.auto}
						bind:value={form[f.key]}
					/>
				{/snippet}
			</Field>
		{/each}
	</form>
	{#snippet footer()}
		<button type="button" class="btn-secondary" disabled={saving} onclick={() => (open = false)}>
			{m.common_cancel()}
		</button>
		<button type="submit" form="{uid}-form" class="btn-primary" disabled={saving}>
			{saving ? m.common_saving() : m.common_save_changes()}
		</button>
	{/snippet}
</Modal>
