<script lang="ts">
	import type { BusinessHours, StoreInfo, StoreInfoInput } from '@atcreations/api-types';
	import { createQuery } from '@tanstack/svelte-query';
	import { untrack } from 'svelte';
	import { fieldErrorMap } from '$lib/api';
	import Field from '$lib/components/Field.svelte';
	import LoadState from '$lib/components/LoadState.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import RequirePermission from '$lib/components/RequirePermission.svelte';
	import Checkbox from '$lib/components/ui/Checkbox.svelte';
	import { formatDateTime } from '$lib/i18n';
	import { toastError } from '$lib/notify';
	import { m } from '$lib/paraglide/messages.js';
	import { createSaveStoreInfoMutation, storeInfoQuery } from '$lib/queries';
	import { getAdminSession } from '$lib/state/session.svelte';
	import { toast } from '$lib/state/toast.svelte';
	import { type Day, normalizeHours, toHoursInput, validateHours } from '$lib/utils/hours';

	const session = getAdminSession();
	const uid = $props.id();

	const infoQuery = createQuery(() => ({
		...storeInfoQuery(),
		enabled: session.can('store:write')
	}));
	const saveInfo = createSaveStoreInfoMutation();
	const info = $derived(infoQuery.data ?? null);
	const saving = $derived(saveInfo.isPending);
	let errors = $state<Record<string, string>>({});

	let email = $state('');
	let phone = $state('');
	let address = $state('');
	let mapUrl = $state('');
	let hours = $state<BusinessHours[]>(normalizeHours([]));

	const dayLabels: Record<Day, () => string> = {
		mon: m.day_mon,
		tue: m.day_tue,
		wed: m.day_wed,
		thu: m.day_thu,
		fri: m.day_fri,
		sat: m.day_sat,
		sun: m.day_sun
	};

	// Bookkeeping for syncing the form with the cache (not rendered, so not reactive).
	let filledFrom: StoreInfo | null = null;
	let baseline = '';
	const formSnapshot = () => JSON.stringify({ email, phone, address, mapUrl, hours });

	function fill(data: StoreInfo) {
		filledFrom = data;
		email = data.email ?? '';
		phone = data.phone ?? '';
		address = data.address ?? '';
		mapUrl = data.mapUrl ?? '';
		hours = normalizeHours(data.businessHours);
		baseline = formSnapshot();
	}

	// Fill the form on first load, and on refetches as long as there are no unsaved edits.
	$effect.pre(() => {
		const next = info;
		if (!next || next === filledFrom) return;
		untrack(() => {
			if (!filledFrom || formSnapshot() === baseline) fill(next);
		});
	});

	const hourErrors = $derived(
		hours.map((h) => {
			const e = validateHours(h);
			return e === 'order'
				? m.store_hours_invalid()
				: e === 'required'
					? m.store_hours_required()
					: '';
		})
	);

	function isHttpsUrl(value: string) {
		try {
			return new URL(value).protocol === 'https:';
		} catch {
			return false;
		}
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (saving) return;
		const e: Record<string, string> = {};
		if (email.trim() && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim()))
			e.email = m.validation_email();
		if (phone.length > 40) e.phone = m.validation_too_long({ max: 40 });
		if (address.length > 300) e.address = m.validation_too_long({ max: 300 });
		if (mapUrl.trim() && (!isHttpsUrl(mapUrl.trim()) || mapUrl.length > 500))
			e.mapUrl = m.validation_url();
		errors = e;
		if (Object.keys(e).length || hourErrors.some(Boolean)) {
			toast.error(m.validation_fix_errors());
			return;
		}
		const body: StoreInfoInput = {
			...(email.trim() ? { email: email.trim() } : {}),
			phone: phone.trim(),
			address: address.trim(),
			...(mapUrl.trim() ? { mapUrl: mapUrl.trim() } : {}),
			businessHours: toHoursInput(hours)
		};
		try {
			fill(await saveInfo.mutateAsync(body));
			toast.success(m.store_saved());
		} catch (error) {
			errors = fieldErrorMap(error);
			toastError(error, m.store_save_failed());
		}
	}
</script>

<RequirePermission permission="store:write">
	<PageHeader
		title={m.store_title()}
		subtitle={info ? m.store_last_updated({ date: formatDateTime(info.updatedAt) }) : undefined}
	/>
	{#if infoQuery.isPending}
		<LoadState loading />
	{:else if infoQuery.isError && !info}
		<LoadState
			error={infoQuery.error}
			fallback={m.store_load_failed()}
			onretry={() => infoQuery.refetch()}
		/>
	{:else}
		<form class="space-y-6" onsubmit={submit} novalidate>
			<fieldset class="space-y-4 card p-5" disabled={saving} aria-labelledby="{uid}-contact">
				<h2 id="{uid}-contact" class="card-title">{m.store_contact()}</h2>
				<div class="grid gap-4 md:grid-cols-2">
					<Field label={m.store_email()} error={errors.email}>
						{#snippet children(control)}
							<input
								{...control}
								type="email"
								class="input"
								autocomplete="email"
								bind:value={email}
							/>
						{/snippet}
					</Field>
					<Field label={m.store_phone()} error={errors.phone}>
						{#snippet children(control)}
							<input
								{...control}
								type="tel"
								class="input"
								maxlength="40"
								autocomplete="tel"
								bind:value={phone}
							/>
						{/snippet}
					</Field>
				</div>
				<Field label={m.store_address()} error={errors.address}>
					{#snippet children(control)}
						<textarea {...control} class="input min-h-20" maxlength="300" bind:value={address}
						></textarea>
					{/snippet}
				</Field>
				<Field label={m.store_map_url()} hint={m.store_map_hint()} error={errors.mapUrl}>
					{#snippet children(control)}
						<input
							{...control}
							type="url"
							class="input"
							maxlength="500"
							placeholder="https://"
							bind:value={mapUrl}
						/>
					{/snippet}
				</Field>
			</fieldset>

			<fieldset class="card p-5" disabled={saving} aria-labelledby="{uid}-hours">
				<h2 id="{uid}-hours" class="mb-4 card-title">{m.store_hours()}</h2>
				<ul class="divide-y divide-zinc-100">
					{#each hours as day, i (day.day)}
						{@const error = hourErrors[i]}
						<li class="flex flex-wrap items-center gap-x-4 gap-y-2 py-3">
							<span class="w-28 font-medium" id="{uid}-{day.day}">{dayLabels[day.day]()}</span>
							<Checkbox label={m.store_closed()} bind:checked={day.closed} />
							{#if !day.closed}
								<label class="flex items-center gap-2 text-sm">
									<span class="text-zinc-600">{m.store_open_time()}</span>
									<input
										type="time"
										class="input w-auto"
										aria-describedby={error ? `${uid}-${day.day}-err` : undefined}
										aria-invalid={error ? 'true' : undefined}
										bind:value={day.open}
									/>
								</label>
								<label class="flex items-center gap-2 text-sm">
									<span class="text-zinc-600">{m.store_close_time()}</span>
									<input
										type="time"
										class="input w-auto"
										aria-describedby={error ? `${uid}-${day.day}-err` : undefined}
										aria-invalid={error ? 'true' : undefined}
										bind:value={day.close}
									/>
								</label>
							{/if}
							{#if error}
								<p id="{uid}-{day.day}-err" class="w-full text-xs text-red-600 sm:w-auto">
									{error}
								</p>
							{/if}
						</li>
					{/each}
				</ul>
				{#if errors.businessHours}<p class="mt-2 text-xs text-red-600">
						{errors.businessHours}
					</p>{/if}
			</fieldset>

			<div class="flex justify-end">
				<button type="submit" class="btn-primary" disabled={saving}>
					{saving ? m.common_saving() : m.common_save_changes()}
				</button>
			</div>
		</form>
	{/if}
</RequirePermission>
