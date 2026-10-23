<script lang="ts">
	import { createQuery, keepPreviousData } from '@tanstack/svelte-query';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { localized } from '$lib/i18n';
	import { safeImageUrl } from '$lib/links';
	import { NAV } from '$lib/nav';
	import { m } from '$lib/paraglide/messages.js';
	import { orderListQuery, productListQuery } from '$lib/queries';
	import { getAdminSession } from '$lib/state/session.svelte';
	import { formatCAD } from '$lib/utils/money';
	import Icon, { type IconName } from './Icon.svelte';
	import StatusBadge from './StatusBadge.svelte';

	let { open = $bindable(false) }: { open?: boolean } = $props();

	const session = getAdminSession();
	const uid = $props.id();

	let dialog = $state<HTMLDialogElement>();
	let input = $state<HTMLInputElement>();
	let query = $state('');
	let debounced = $state('');
	let activeIndex = $state(0);
	let returnFocus: HTMLElement | null = null;

	$effect(() => {
		const value = query.trim();
		const timer = setTimeout(() => (debounced = value), 180);
		return () => clearTimeout(timer);
	});

	$effect(() => {
		const el = dialog;
		if (!el) return;
		if (open && !el.open) {
			returnFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
			el.showModal();
			input?.focus();
		} else if (!open && el.open) {
			el.close();
		}
	});

	const searching = $derived(debounced.length >= 2);
	const products = createQuery(() => ({
		...productListQuery({ q: debounced, pageSize: 5 }),
		enabled: open && searching,
		placeholderData: keepPreviousData
	}));
	const orders = createQuery(() => ({
		...orderListQuery({ q: debounced, pageSize: 5 }),
		enabled: open && searching && session.can('orders:read'),
		placeholderData: keepPreviousData
	}));

	interface Item {
		id: string;
		group: string;
		label: string;
		hint?: string;
		icon?: IconName;
		image?: string;
		status?: { kind: 'product' | 'order'; value: string };
		href: string;
	}

	const pageItems = $derived<Item[]>(
		NAV.flat()
			.filter((i) => !i.permission || session.can(i.permission))
			.map((i) => ({
				id: `page:${i.href}`,
				group: m.cmd_pages(),
				label: i.label(),
				icon: i.icon,
				href: resolve(i.href)
			}))
	);

	const actionItems = $derived<Item[]>(
		(
			[
				session.can('products:write') && {
					id: 'action:new-product',
					group: m.cmd_actions(),
					label: m.dash_add_product(),
					icon: 'plus' as const,
					href: resolve('/products/new')
				},
				session.can('staff:manage') && {
					id: 'action:invite',
					group: m.cmd_actions(),
					label: m.dash_invite_staff(),
					icon: 'mail' as const,
					href: `${resolve('/staff')}?invite=1`
				},
				session.can('orders:read') && {
					id: 'action:pending',
					group: m.cmd_actions(),
					label: m.cmd_review_pending(),
					icon: 'inbox' as const,
					href: `${resolve('/orders')}?status=pending`
				}
			] as (Item | false)[]
		).filter((x): x is Item => !!x)
	);

	function matches(label: string) {
		const q = query.trim().toLocaleLowerCase();
		return !q || label.toLocaleLowerCase().includes(q);
	}

	const items = $derived.by<Item[]>(() => {
		const out = [...pageItems, ...actionItems].filter((i) => matches(i.label));
		if (searching) {
			for (const p of products.data?.data ?? []) {
				out.push({
					id: `product:${p.id}`,
					group: m.cmd_products(),
					label: localized(p.name),
					hint: p.finalPriceCents === null ? undefined : formatCAD(p.finalPriceCents),
					image: safeImageUrl(p.images[0]?.url),
					icon: 'box',
					status: { kind: 'product', value: p.status },
					href: resolve(`/products/${p.id}`)
				});
			}
			for (const o of orders.data?.data ?? []) {
				out.push({
					id: `order:${o.id}`,
					group: m.cmd_orders(),
					label: `${o.number} · ${o.customer.name}`,
					hint: formatCAD(o.totalCents),
					icon: 'inbox',
					status: { kind: 'order', value: o.status },
					href: resolve(`/orders/${o.id}`)
				});
			}
		}
		return out;
	});

	const groups = $derived.by(() => {
		const out: [string, { item: Item; index: number }[]][] = [];
		items.forEach((item, index) => {
			let group = out.find(([name]) => name === item.group);
			if (!group) out.push((group = [item.group, []]));
			group[1].push({ item, index });
		});
		return out;
	});

	const loading = $derived(
		searching && (products.isFetching || orders.isFetching) && items.length === 0
	);

	$effect(() => {
		// Reset the highlight whenever the result set changes.
		void items.length;
		void debounced;
		activeIndex = 0;
	});

	function select(item: Item | undefined) {
		if (!item) return;
		open = false;
		// Hrefs are built with resolve() above.
		// eslint-disable-next-line svelte/no-navigation-without-resolve
		void goto(item.href);
	}

	function onkeydown(event: KeyboardEvent) {
		if (event.key === 'ArrowDown') {
			event.preventDefault();
			activeIndex = items.length ? (activeIndex + 1) % items.length : 0;
		} else if (event.key === 'ArrowUp') {
			event.preventDefault();
			activeIndex = items.length ? (activeIndex - 1 + items.length) % items.length : 0;
		} else if (event.key === 'Enter') {
			event.preventDefault();
			select(items[activeIndex]);
		}
	}

	$effect(() => {
		if (!open) return;
		document.getElementById(`${uid}-opt-${activeIndex}`)?.scrollIntoView({ block: 'nearest' });
	});

	function handleClose() {
		open = false;
		query = '';
		debounced = '';
		const target = returnFocus;
		returnFocus = null;
		if (target?.isConnected) target.focus();
	}
</script>

<svelte:window
	onkeydown={(event) => {
		const typing =
			event.target instanceof HTMLElement &&
			(event.target.isContentEditable || /^(INPUT|TEXTAREA|SELECT)$/.test(event.target.tagName));
		if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
			event.preventDefault();
			open = !open;
		} else if (event.key === '/' && !typing && !open) {
			event.preventDefault();
			open = true;
		}
	}}
/>

<dialog
	bind:this={dialog}
	aria-label={m.cmd_label()}
	class="mx-auto mt-[12vh] w-[calc(100%-2rem)] max-w-xl animate-pop-in overflow-hidden rounded-xl bg-white p-0 text-zinc-900 shadow-popover"
	onclose={handleClose}
	onclick={(event) => {
		if (event.target === dialog) open = false;
	}}
>
	{#if open}
		<div class="flex items-center gap-3 border-b border-zinc-200 px-4">
			<Icon name="search" class="text-zinc-400" />
			<input
				bind:this={input}
				bind:value={query}
				type="text"
				role="combobox"
				aria-expanded="true"
				aria-controls="{uid}-list"
				aria-activedescendant={items.length ? `${uid}-opt-${activeIndex}` : undefined}
				aria-autocomplete="list"
				class="h-13 flex-1 bg-transparent text-[15px] placeholder:text-zinc-400 focus:outline-none"
				placeholder={m.cmd_placeholder()}
				maxlength="100"
				autocomplete="off"
				spellcheck="false"
				{onkeydown}
			/>
			{#if searching && (products.isFetching || orders.isFetching)}
				<span
					class="size-4 animate-spin rounded-full border-2 border-zinc-200 border-t-brand-600"
					aria-hidden="true"
				></span>
			{/if}
			<kbd class="kbd">Esc</kbd>
		</div>

		<div
			id="{uid}-list"
			role="listbox"
			aria-label={m.cmd_label()}
			class="max-h-[50vh] overflow-y-auto p-2"
		>
			{#if loading}
				<p class="px-3 py-8 text-center text-sm text-zinc-500" role="status">
					{m.cmd_searching()}
				</p>
			{:else if items.length === 0}
				<p class="px-3 py-8 text-center text-sm text-zinc-500">
					{m.cmd_no_results()}
				</p>
			{:else}
				{#each groups as [group, entries], gi (group)}
					<div role="group" aria-labelledby="{uid}-g-{gi}" class="mb-1 last:mb-0">
						<p id="{uid}-g-{gi}" class="px-3 pt-2 pb-1 text-xs font-medium text-zinc-400">
							{group}
						</p>
						{#each entries as { item, index } (item.id)}
							<div
								id="{uid}-opt-{index}"
								role="option"
								tabindex="-1"
								aria-selected={index === activeIndex}
								class={[
									'flex cursor-pointer items-center gap-3 rounded-lg px-3 py-2 text-sm',
									index === activeIndex ? 'bg-brand-100 text-brand-900' : 'text-zinc-700'
								]}
								onclick={() => select(item)}
								onkeydown={() => {}}
								onpointermove={() => (activeIndex = index)}
							>
								{#if item.image}
									<img src={item.image} alt="" class="size-7 rounded object-cover" />
								{:else if item.icon}
									<span
										class="flex size-7 items-center justify-center rounded-md border border-zinc-200 bg-white text-zinc-500"
									>
										<Icon name={item.icon} size={15} />
									</span>
								{/if}
								<span class="min-w-0 flex-1 truncate font-medium">{item.label}</span>
								{#if item.status}
									{#if item.status.kind === 'product'}
										<StatusBadge kind="product" value={item.status.value as 'draft'} />
									{:else}
										<StatusBadge kind="order" value={item.status.value as 'pending'} />
									{/if}
								{/if}
								{#if item.hint}<span class="text-xs text-zinc-500 tabular-nums">{item.hint}</span
									>{/if}
								{#if index === activeIndex}
									<Icon name="corner" size={14} class="text-brand-500" />
								{/if}
							</div>
						{/each}
					</div>
				{/each}
			{/if}
		</div>
	{/if}
</dialog>
