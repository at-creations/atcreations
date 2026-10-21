<script lang="ts">
	import type { Order, OrderEvent, OrderStatus } from '@atcreations/api-types';
	import { tick, untrack } from 'svelte';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { api, ApiError, unwrap } from '$lib/api';
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
	import { getAdminSession } from '$lib/state/session.svelte';
	import { toast } from '$lib/state/toast.svelte';
	import { formatCAD } from '$lib/utils/money';
	import { allowedNextStatuses, isShippingEditable } from '$lib/utils/order-status';

	const session = getAdminSession();
	const uid = $props.id();

	let order = $state<Order | null>(null);
	let loading = $state(true);
	let loadError = $state<unknown>(null);
	let notFound = $state(false);

	let staffNote = $state('');
	let savingNote = $state(false);
	let statusOpen = $state(false);
	let statusTarget = $state<OrderStatus | null>(null);
	let statusDialog = $state<ReturnType<typeof StatusDialog>>();
	let addressOpen = $state(false);
	let addressDialog = $state<ReturnType<typeof AddressDialog>>();

	function setOrder(next: Order) {
		order = next;
		staffNote = next.staffNote ?? '';
	}

	async function load(id: string) {
		loading = true;
		loadError = null;
		notFound = false;
		try {
			const res = await unwrap(api().GET('/v1/admin/orders/{id}', { params: { path: { id } } }));
			setOrder(res.data);
		} catch (error) {
			if (error instanceof ApiError && (error.status === 404 || error.status === 400))
				notFound = true;
			else loadError = error;
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		const id = page.params.id ?? '';
		if (!session.can('orders:read')) return;
		untrack(() => load(id));
	});

	const nextStatuses = $derived(order ? allowedNextStatuses(order.status) : []);
	const canWrite = $derived(session.can('orders:write'));
	const canShipping = $derived(session.can('orders:shipping'));
	const shippingEditable = $derived(order ? isShippingEditable(order.status) : false);
	const noteDirty = $derived(!!order && staffNote !== (order.staffNote ?? ''));

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
		savingNote = true;
		try {
			const res = await unwrap(
				api().PUT('/v1/admin/orders/{id}/notes', {
					params: { path: { id: order.id } },
					body: { staffNote: staffNote.slice(0, 5000) }
				})
			);
			setOrder(res.data);
			toast.success(m.od_note_saved());
		} catch (error) {
			toastError(error, m.od_note_failed());
		} finally {
			savingNote = false;
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
	<a
		href={resolve('/orders')}
		class="mb-4 inline-flex items-center gap-1 text-sm text-slate-600 hover:underline"
	>
		<Icon name="left" size={16} />{m.od_back()}
	</a>

	{#if loading && !order}
		<LoadState loading />
	{:else if notFound}
		<p class="card p-10 text-center text-slate-600">{m.od_not_found()}</p>
	{:else if loadError || !order}
		<LoadState error={loadError} onretry={() => load(page.params.id ?? '')} />
	{:else}
		<PageHeader
			title={m.od_title({ number: order.number })}
			subtitle={formatDateTime(order.createdAt)}
		>
			{#snippet actions()}
				<StatusBadge kind="order" value={order!.status} />
			{/snippet}
		</PageHeader>

		<div class="grid gap-6 lg:grid-cols-3">
			<div class="space-y-6 lg:col-span-2">
				<!-- Items & totals -->
				<section class="overflow-hidden card" aria-labelledby="{uid}-items">
					<h2 id="{uid}-items" class="border-b border-slate-200 px-5 py-3 font-semibold">
						{m.od_items()}
					</h2>
					<ul class="divide-y divide-slate-100">
						{#each order.items as item (item.productId)}
							{@const src = safeImageUrl(item.imageUrl)}
							<li class="flex items-center gap-3 px-5 py-3">
								{#if src}
									<img {src} alt="" class="size-14 rounded-lg object-cover" loading="lazy" />
								{:else}
									<span class="flex size-14 items-center justify-center rounded-lg bg-slate-100">
										<Icon name="image" class="text-slate-400" />
									</span>
								{/if}
								<div class="min-w-0 flex-1">
									<a
										href={resolve(`/products/${item.productId}`)}
										class="block truncate font-medium hover:underline">{localized(item.name)}</a
									>
									<p class="text-xs text-slate-500">
										{m.od_unit_price()}: {formatCAD(item.unitPriceCents)}
										{#if item.discountPercent > 0}
											<span class="ml-1 badge bg-red-100 text-red-700"
												>{m.products_discount({ percent: item.discountPercent })}</span
											>
										{/if}
										· {m.od_qty()}: {item.quantity}
									</p>
								</div>
								<span class="font-medium">{formatCAD(item.lineTotalCents)}</span>
							</li>
						{/each}
					</ul>
					<dl class="space-y-1 border-t border-slate-200 bg-slate-50 px-5 py-4 text-sm">
						<div class="flex justify-between">
							<dt>{m.od_subtotal()}</dt>
							<dd>{formatCAD(order.subtotalCents)}</dd>
						</div>
						{#if order.discountCents > 0}
							<div class="flex justify-between text-red-700">
								<dt>{m.od_discount()}</dt>
								<dd>−{formatCAD(order.discountCents)}</dd>
							</div>
						{/if}
						<div class="flex justify-between">
							<dt>{m.od_shipping_cost()}</dt>
							<dd>{formatCAD(order.shippingCents)}</dd>
						</div>
						<div class="flex justify-between">
							<dt>{m.od_tax()}</dt>
							<dd>{formatCAD(order.taxCents)}</dd>
						</div>
						<div
							class="flex justify-between border-t border-slate-200 pt-2 text-base font-semibold"
						>
							<dt>{m.od_total()}</dt>
							<dd>{formatCAD(order.totalCents)}</dd>
						</div>
					</dl>
				</section>

				<!-- Status transitions -->
				{#if canWrite}
					<section class="card p-5" aria-labelledby="{uid}-status">
						<h2 id="{uid}-status" class="mb-3 font-semibold">{m.od_change_status()}</h2>
						{#if order.trackingNumber}
							<p class="mb-3 text-sm">
								{m.od_tracking()}: <span class="font-mono select-all">{order.trackingNumber}</span>
							</p>
						{/if}
						{#if nextStatuses.length === 0}
							<p class="text-sm text-slate-500">{m.od_no_transitions()}</p>
						{:else}
							<div class="flex flex-wrap gap-2">
								{#each nextStatuses as next (next)}
									<button
										type="button"
										class={next === 'cancelled' || next === 'refunded'
											? 'btn-danger'
											: 'btn-primary'}
										onclick={() => openStatus(next)}
									>
										{m.od_mark_as({ status: orderStatusLabel(next) })}
									</button>
								{/each}
							</div>
						{/if}
					</section>
				{/if}

				<!-- Notes -->
				{#if order.customerNote}
					<section class="card p-5">
						<h2 class="mb-2 font-semibold">{m.od_customer_note()}</h2>
						<p class="text-sm whitespace-pre-line text-slate-700">{order.customerNote}</p>
					</section>
				{/if}
				<section class="card p-5">
					<form onsubmit={saveNote}>
						<label for="{uid}-note" class="mb-1 block font-semibold">{m.od_staff_note()}</label>
						<p id="{uid}-note-hint" class="mb-2 text-xs text-slate-500">{m.od_staff_note_hint()}</p>
						<textarea
							id="{uid}-note"
							class="input min-h-28"
							maxlength="5000"
							aria-describedby="{uid}-note-hint"
							readonly={!canWrite}
							bind:value={staffNote}></textarea>
						{#if canWrite}
							<div class="mt-2 flex justify-end">
								<button type="submit" class="btn-primary" disabled={!noteDirty || savingNote}>
									{savingNote ? m.common_saving() : m.common_save()}
								</button>
							</div>
						{/if}
					</form>
				</section>
			</div>

			<div class="space-y-6">
				<section class="card p-5">
					<h2 class="mb-3 font-semibold">{m.od_customer()}</h2>
					<div class="space-y-1 text-sm">
						<p class="font-medium">{order.customer.name}</p>
						<p class="break-all">
							<span class="sr-only">{m.od_email()}:</span>
							<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- mailto link -->
							<a class="text-brand-700 hover:underline" href="mailto:{order.customer.email}"
								>{order.customer.email}</a
							>
						</p>
						{#if order.customer.phone}
							<p><span class="sr-only">{m.od_phone()}:</span>{order.customer.phone}</p>
						{/if}
					</div>
				</section>

				<section class="card p-5">
					<div class="mb-3 flex items-center justify-between gap-2">
						<h2 class="font-semibold">{m.od_shipping()}</h2>
						{#if canShipping}
							<button
								type="button"
								class="btn-secondary btn-sm"
								disabled={!shippingEditable}
								onclick={openAddress}
							>
								<Icon name="edit" size={14} />{m.od_edit_address()}
							</button>
						{/if}
					</div>
					<address class="text-sm leading-relaxed not-italic">
						{order.shipping.name}<br />
						{order.shipping.line1}<br />
						{#if order.shipping.line2}{order.shipping.line2}<br />{/if}
						{order.shipping.city}{order.shipping.region ? `, ${order.shipping.region}` : ''}
						{order.shipping.postalCode}<br />
						{order.shipping.country}<br />
						{order.shipping.phone}
					</address>
					{#if canShipping && !shippingEditable}
						<p class="mt-2 text-xs text-slate-500">{m.od_address_locked()}</p>
					{/if}
				</section>

				<section class="card p-5" aria-labelledby="{uid}-history">
					<h2 id="{uid}-history" class="mb-3 font-semibold">{m.od_history()}</h2>
					<ol class="relative space-y-4 border-l border-slate-200 pl-4">
						{#each history as event, i (i)}
							<li class="relative">
								<span
									class="absolute top-1.5 -left-[21px] size-2.5 rounded-full bg-brand-800 ring-4 ring-white"
								></span>
								<p class="text-sm font-medium">{eventText(event)}</p>
								{#if event.note}<p class="text-sm whitespace-pre-line text-slate-600">
										{event.note}
									</p>{/if}
								<p class="text-xs text-slate-500">
									<time datetime={event.at}>{formatDateTime(event.at)}</time>
									· {m.od_by({ actor: event.actor })}
								</p>
							</li>
						{/each}
					</ol>
				</section>
			</div>
		</div>

		<StatusDialog
			bind:this={statusDialog}
			bind:open={statusOpen}
			{order}
			target={statusTarget}
			onupdated={setOrder}
		/>
		<AddressDialog bind:this={addressDialog} bind:open={addressOpen} {order} onupdated={setOrder} />
	{/if}
</RequirePermission>
