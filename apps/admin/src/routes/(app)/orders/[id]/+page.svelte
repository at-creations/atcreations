<script lang="ts">
	import type { OrderEvent, OrderStatus } from '@atcreations/api-types';
	import { createQuery } from '@tanstack/svelte-query';
	import { tick } from 'svelte';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { ApiError } from '$lib/api';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import Icon from '$lib/components/Icon.svelte';
	import LoadState from '$lib/components/LoadState.svelte';
	import AddressDialog from '$lib/components/orders/AddressDialog.svelte';
	import StatusDialog from '$lib/components/orders/StatusDialog.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import RequirePermission from '$lib/components/RequirePermission.svelte';
	import StatusBadge from '$lib/components/StatusBadge.svelte';
	import { formatDateTime, localized, orderStatusLabel } from '$lib/i18n';
	import { safeImageUrl } from '$lib/links';
	import { toastError } from '$lib/notify';
	import { m } from '$lib/paraglide/messages.js';
	import { createOrderNoteMutation, orderDetailQuery } from '$lib/queries';
	import { getAdminSession } from '$lib/state/session.svelte';
	import { toast } from '$lib/state/toast.svelte';
	import { formatCAD } from '$lib/utils/money';
	import { allowedNextStatuses, isShippingEditable } from '$lib/utils/order-status';

	const session = getAdminSession();
	const uid = $props.id();

	const orderQuery = createQuery(() => ({
		...orderDetailQuery(page.params.id ?? ''),
		enabled: session.can('orders:read')
	}));
	const saveNoteMutation = createOrderNoteMutation();

	const order = $derived(orderQuery.data ?? null);
	const notFound = $derived(
		orderQuery.error instanceof ApiError &&
			(orderQuery.error.status === 404 || orderQuery.error.status === 400)
	);
	const loadError = $derived(!order && orderQuery.isError ? orderQuery.error : null);

	// Editable copy of the note; reset whenever the server value changes.
	const serverNote = $derived(order?.staffNote ?? '');
	let staffNote = $derived(serverNote);
	const savingNote = $derived(saveNoteMutation.isPending);
	let statusOpen = $state(false);
	let statusTarget = $state<OrderStatus | null>(null);
	let statusDialog = $state<ReturnType<typeof StatusDialog>>();
	let addressOpen = $state(false);
	let addressDialog = $state<ReturnType<typeof AddressDialog>>();

	const nextStatuses = $derived(order ? allowedNextStatuses(order.status) : []);
	/** The "happy path" next step, offered as the primary action. */
	const primaryNext = $derived(
		nextStatuses.find((s) => s !== 'cancelled' && s !== 'refunded') ?? null
	);
	/** Fulfilment steps shown in the progress tracker. */
	const FLOW: readonly OrderStatus[] = [
		'pending',
		'confirmed',
		'processing',
		'shipped',
		'delivered',
		'completed'
	];
	const flowIndex = $derived(order ? FLOW.indexOf(order.status) : -1);
	const canWrite = $derived(session.can('orders:write'));
	const canShipping = $derived(session.can('orders:shipping'));
	const shippingEditable = $derived(order ? isShippingEditable(order.status) : false);
	const noteDirty = $derived(!!order && staffNote !== serverNote);

	async function openStatus(target: OrderStatus) {
		statusTarget = target;
		await tick();
		statusDialog?.reset();
		statusOpen = true;
	}

	async function openAddress() {
		addressDialog?.reset();
		addressOpen = true;
	}

	async function saveNote(event: SubmitEvent) {
		event.preventDefault();
		if (!order || savingNote) return;
		try {
			await saveNoteMutation.mutateAsync({ id: order.id, staffNote: staffNote.slice(0, 5000) });
			toast.success(m.od_note_saved());
		} catch (error) {
			toastError(error, m.od_note_failed());
		}
	}

	function eventText(e: OrderEvent): string {
		switch (e.type) {
			case 'created':
				return m.od_event_created();
			case 'status_changed':
				return m.od_event_status_changed({
					from: e.fromStatus ? orderStatusLabel(e.fromStatus) : '—',
					to: e.toStatus ? orderStatusLabel(e.toStatus) : '—'
				});
			case 'note_updated':
				return m.od_event_note_updated();
			case 'shipping_updated':
				return m.od_event_shipping_updated();
			default:
				return m.od_event_customer_notified();
		}
	}

	const history = $derived(
		order ? [...order.history].sort((a, b) => b.at.localeCompare(a.at)) : []
	);
</script>

<RequirePermission permission="orders:read">
	{#if notFound}
		<div class="card">
			<EmptyState icon="inbox" title={m.od_not_found()}>
				<a class="btn-secondary" href={resolve('/orders')}
					><Icon name="left" size={16} />{m.od_back()}</a
				>
			</EmptyState>
		</div>
	{:else if loadError}
		<div class="card"><LoadState error={loadError} onretry={() => orderQuery.refetch()} /></div>
	{:else if !order}
		<div class="space-y-6" role="status" aria-label={m.common_loading()}>
			<div class="h-8 w-56 animate-pulse rounded-md bg-zinc-200"></div>
			<div class="grid gap-6 lg:grid-cols-3">
				<div class="h-96 animate-pulse card lg:col-span-2"></div>
				<div class="h-64 animate-pulse card"></div>
			</div>
		</div>
	{:else}
		<PageHeader
			title={m.od_title({ number: order.number })}
			subtitle={formatDateTime(order.createdAt)}
			back={{ href: resolve('/orders'), label: m.od_back() }}
		>
			{#snippet meta()}
				<StatusBadge kind="order" value={order!.status} />
			{/snippet}
			{#snippet actions()}
				{#if canWrite}
					{#each nextStatuses as next (next)}
						{@const danger = next === 'cancelled' || next === 'refunded'}
						<button
							type="button"
							class={danger
								? 'btn-secondary text-red-600 hover:border-red-300 hover:bg-red-50 hover:text-red-700'
								: next === primaryNext
									? 'btn-primary'
									: 'btn-secondary'}
							onclick={() => openStatus(next)}
						>
							{#if !danger && next === primaryNext}<Icon name="check" size={16} />{/if}
							{m.od_mark_as({ status: orderStatusLabel(next) })}
						</button>
					{/each}
				{/if}
			{/snippet}
		</PageHeader>

		<div class="grid items-start gap-6 lg:grid-cols-3">
			<div class="min-w-0 space-y-6 lg:col-span-2">
				<!-- Progress -->
				<section class="card p-5" aria-label={m.od_progress()}>
					{#if order.status === 'cancelled' || order.status === 'refunded'}
						<p class="flex items-center gap-3 text-sm">
							<span
								class="flex size-8 items-center justify-center rounded-full bg-red-50 text-red-600"
							>
								<Icon name="ban" size={16} />
							</span>
							<span class="font-medium text-zinc-900"
								>{m.od_closed({ status: orderStatusLabel(order.status) })}</span
							>
						</p>
					{:else}
						<ol class="flex items-center">
							{#each FLOW as step, i (step)}
								{@const done = i <= flowIndex}
								<li class={['flex items-center', i < FLOW.length - 1 && 'flex-1']}>
									<span class="flex flex-col items-center gap-1.5">
										<span
											class={[
												'flex size-7 items-center justify-center rounded-full text-xs font-semibold',
												done
													? 'bg-brand-200 text-brand-900'
													: 'bg-white text-zinc-400 ring-1 ring-zinc-300'
											]}
											aria-current={i === flowIndex ? 'step' : undefined}
										>
											{#if i < flowIndex}<Icon name="check" size={14} />{:else}{i + 1}{/if}
										</span>
										<span
											class={[
												'hidden text-xs whitespace-nowrap sm:block',
												i === flowIndex ? 'font-semibold text-zinc-900' : 'text-zinc-500'
											]}>{orderStatusLabel(step)}</span
										>
									</span>
									{#if i < FLOW.length - 1}
										<span
											class={[
												'mx-2 h-0.5 flex-1 rounded sm:mb-5',
												i < flowIndex ? 'bg-brand-300' : 'bg-zinc-200'
											]}
											aria-hidden="true"
										></span>
									{/if}
								</li>
							{/each}
						</ol>
					{/if}
					{#if order.trackingNumber}
						<p
							class="mt-4 flex items-center gap-2 border-t border-zinc-100 pt-4 text-sm text-zinc-600"
						>
							{m.od_tracking()}:
							<span class="font-mono font-medium text-zinc-900 select-all"
								>{order.trackingNumber}</span
							>
						</p>
					{/if}
				</section>

				<!-- Items & totals -->
				<section class="card" aria-labelledby="{uid}-items">
					<div class="card-header">
						<h2 id="{uid}-items" class="card-title">{m.od_items()}</h2>
						<span class="text-xs text-zinc-500 tabular-nums"
							>{m.od_item_count({ count: order.items.reduce((n, i) => n + i.quantity, 0) })}</span
						>
					</div>
					<ul class="divide-y divide-zinc-100">
						{#each order.items as item (item.productId)}
							{@const src = safeImageUrl(item.imageUrl)}
							<li class="flex items-center gap-4 px-5 py-3">
								{#if src}
									<img
										{src}
										alt=""
										class="size-12 rounded-lg object-cover ring-1 ring-zinc-200"
										loading="lazy"
									/>
								{:else}
									<span class="flex size-12 items-center justify-center rounded-lg bg-zinc-100">
										<Icon name="image" class="text-zinc-400" />
									</span>
								{/if}
								<div class="min-w-0 flex-1">
									<a
										href={resolve(`/products/${item.productId}`)}
										class="block truncate text-sm font-medium text-zinc-900 hover:text-brand-700"
										>{localized(item.name)}</a
									>
									<p class="text-xs text-zinc-500 tabular-nums">
										{formatCAD(item.unitPriceCents)} × {item.quantity}
										{#if item.discountPercent > 0}
											<span class="ml-1 badge bg-red-50 text-red-700"
												>{m.products_discount({ percent: item.discountPercent })}</span
											>
										{/if}
									</p>
								</div>
								<span class="text-sm font-medium text-zinc-900 tabular-nums"
									>{formatCAD(item.lineTotalCents)}</span
								>
							</li>
						{/each}
					</ul>
					<dl
						class="space-y-1.5 rounded-b-xl border-t border-zinc-200 bg-zinc-50/70 px-5 py-4 text-sm tabular-nums"
					>
						<div class="flex justify-between text-zinc-600">
							<dt>{m.od_subtotal()}</dt>
							<dd>{formatCAD(order.subtotalCents)}</dd>
						</div>
						{#if order.discountCents > 0}
							<div class="flex justify-between text-red-700">
								<dt>{m.od_discount()}</dt>
								<dd>−{formatCAD(order.discountCents)}</dd>
							</div>
						{/if}
						<div class="flex justify-between text-zinc-600">
							<dt>{m.od_shipping_cost()}</dt>
							<dd>{formatCAD(order.shippingCents)}</dd>
						</div>
						<div class="flex justify-between text-zinc-600">
							<dt>{m.od_tax()}</dt>
							<dd>{formatCAD(order.taxCents)}</dd>
						</div>
						<div
							class="flex justify-between border-t border-zinc-200 pt-2 text-base font-semibold text-zinc-900"
						>
							<dt>{m.od_total()}</dt>
							<dd>{formatCAD(order.totalCents)}</dd>
						</div>
					</dl>
				</section>

				<!-- Notes -->
				{#if order.customerNote}
					<section class="card">
						<div class="card-header">
							<h2 class="card-title">{m.od_customer_note()}</h2>
						</div>
						<p class="p-5 text-sm whitespace-pre-line text-zinc-700">{order.customerNote}</p>
					</section>
				{/if}
				<section class="card">
					<form onsubmit={saveNote}>
						<div class="card-header">
							<label for="{uid}-note" class="card-title">{m.od_staff_note()}</label>
							<span id="{uid}-note-hint" class="text-xs text-zinc-500"
								>{m.od_staff_note_hint()}</span
							>
						</div>
						<div class="p-5">
							<textarea
								id="{uid}-note"
								class="input h-auto min-h-28 py-2"
								maxlength="5000"
								aria-describedby="{uid}-note-hint"
								readonly={!canWrite}
								bind:value={staffNote}></textarea>
							{#if canWrite && noteDirty}
								<div class="mt-3 flex justify-end gap-2">
									<button
										type="button"
										class="btn-secondary"
										disabled={savingNote}
										onclick={() => (staffNote = serverNote)}>{m.pf_discard()}</button
									>
									<button type="submit" class="btn-primary" disabled={savingNote}>
										{savingNote ? m.common_saving() : m.common_save()}
									</button>
								</div>
							{/if}
						</div>
					</form>
				</section>
			</div>

			<div class="min-w-0 space-y-6">
				<section class="card">
					<div class="card-header">
						<h2 class="card-title">{m.od_customer()}</h2>
					</div>
					<div class="space-y-2 p-5 text-sm">
						<p class="flex items-center gap-3 font-medium text-zinc-900">
							<span
								class="flex size-9 items-center justify-center rounded-full bg-zinc-100 text-xs font-semibold text-zinc-600 uppercase"
								aria-hidden="true">{order.customer.name.slice(0, 2)}</span
							>
							{order.customer.name}
						</p>
						<p class="flex items-center gap-2 break-all text-zinc-600">
							<Icon name="mail" size={14} class="text-zinc-400" />
							<span class="sr-only">{m.od_email()}:</span>
							<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- mailto link -->
							<a class="text-brand-600 hover:underline" href="mailto:{order.customer.email}"
								>{order.customer.email}</a
							>
						</p>
						{#if order.customer.phone}
							<p class="text-zinc-600">
								<span class="sr-only">{m.od_phone()}:</span>{order.customer.phone}
							</p>
						{/if}
					</div>
				</section>

				<section class="card">
					<div class="card-header">
						<h2 class="card-title">{m.od_shipping()}</h2>
						{#if canShipping && shippingEditable}
							<button type="button" class="btn-ghost btn-sm" onclick={openAddress}>
								<Icon name="edit" size={14} />{m.common_edit()}
							</button>
						{/if}
					</div>
					<div class="p-5">
						<address class="text-sm leading-relaxed text-zinc-700 not-italic">
							<span class="font-medium text-zinc-900">{order.shipping.name}</span><br />
							{order.shipping.line1}<br />
							{#if order.shipping.line2}{order.shipping.line2}<br />{/if}
							{order.shipping.city}{order.shipping.region ? `, ${order.shipping.region}` : ''}
							{order.shipping.postalCode}<br />
							{order.shipping.country}<br />
							{order.shipping.phone}
						</address>
						{#if canShipping && !shippingEditable}
							<p class="mt-3 text-xs text-zinc-500">{m.od_address_locked()}</p>
						{/if}
					</div>
				</section>

				<section class="card" aria-labelledby="{uid}-history">
					<div class="card-header">
						<h2 id="{uid}-history" class="card-title">{m.od_history()}</h2>
					</div>
					<ol class="relative space-y-5 p-5">
						{#each history as event, i (i)}
							<li class="relative pl-6">
								{#if i < history.length - 1}
									<span
										class="absolute top-4 bottom-[-1.25rem] left-[5px] w-px bg-zinc-200"
										aria-hidden="true"
									></span>
								{/if}
								<span
									class={[
										'absolute top-1.5 left-0 size-[11px] rounded-full ring-4 ring-white',
										i === 0 ? 'bg-brand-400' : 'bg-zinc-300'
									]}
									aria-hidden="true"
								></span>
								<p class="text-sm font-medium text-zinc-900">{eventText(event)}</p>
								{#if event.note}
									<p class="mt-0.5 text-sm whitespace-pre-line text-zinc-600">{event.note}</p>
								{/if}
								<p class="mt-0.5 text-xs text-zinc-500">
									<time datetime={event.at}>{formatDateTime(event.at)}</time>
									· {m.od_by({ actor: event.actor })}
								</p>
							</li>
						{/each}
					</ol>
				</section>
			</div>
		</div>

		<StatusDialog bind:this={statusDialog} bind:open={statusOpen} {order} target={statusTarget} />
		<AddressDialog bind:this={addressDialog} bind:open={addressOpen} {order} />
	{/if}
</RequirePermission>
