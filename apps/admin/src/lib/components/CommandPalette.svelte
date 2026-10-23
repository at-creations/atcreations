<script lang="ts">
	import { createQuery, keepPreviousData } from '@tanstack/svelte-query';
	import { Command, Dialog } from 'bits-ui';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { localized } from '$lib/i18n';
	import { NAV } from '$lib/nav';
	import { m } from '$lib/paraglide/messages.js';
	import { orderListQuery, productListQuery } from '$lib/queries';
	import { getAdminSession } from '$lib/state/session.svelte';
	import { formatCAD } from '$lib/utils/money';
	import Icon, { type IconName } from './Icon.svelte';
	import StatusBadge from './StatusBadge.svelte';
	import Thumb from './ui/Thumb.svelte';

	let { open = $bindable(false) }: { open?: boolean } = $props();

	const session = getAdminSession();

	let query = $state('');
	let debounced = $state('');

	$effect(() => {
		const value = query.trim();
		const timer = setTimeout(() => (debounced = value), 180);
		return () => clearTimeout(timer);
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
		label: string;
		hint?: string;
		icon?: IconName;
		image?: string;
		status?:
			| { kind: 'product'; value: 'draft' | 'published' | 'archived' }
			| { kind: 'order'; value: string };
		href: string;
	}

	function matches(label: string) {
		const q = query.trim().toLocaleLowerCase();
		return !q || label.toLocaleLowerCase().includes(q);
	}

	const pageItems = $derived<Item[]>(
		NAV.flat()
			.filter((i) => !i.permission || session.can(i.permission))
			.map((i) => ({ id: `page:${i.href}`, label: i.label(), icon: i.icon, href: resolve(i.href) }))
			.filter((i) => matches(i.label))
	);

	const actionItems = $derived<Item[]>(
		(
			[
				session.can('products:write') && {
					id: 'action:new-product',
					label: m.dash_add_product(),
					icon: 'plus' as const,
					href: resolve('/products/new')
				},
				session.can('staff:manage') && {
					id: 'action:invite',
					label: m.dash_invite_staff(),
					icon: 'mail' as const,
					href: `${resolve('/staff')}?invite=1`
				},
				session.can('orders:read') && {
					id: 'action:pending',
					label: m.cmd_review_pending(),
					icon: 'inbox' as const,
					href: `${resolve('/orders')}?status=pending`
				}
			] as (Item | false)[]
		)
			.filter((x): x is Item => !!x)
			.filter((i) => matches(i.label))
	);

	const productItems = $derived<Item[]>(
		searching
			? (products.data?.data ?? []).map((p) => ({
					id: `product:${p.id}`,
					label: localized(p.name),
					hint: p.finalPriceCents === null ? undefined : formatCAD(p.finalPriceCents),
					image: p.images[0]?.url,
					status: { kind: 'product' as const, value: p.status },
					href: resolve(`/products/${p.id}`)
				}))
			: []
	);

	const orderItems = $derived<Item[]>(
		searching
			? (orders.data?.data ?? []).map((o) => ({
					id: `order:${o.id}`,
					label: `${o.number} · ${o.customer.name}`,
					hint: formatCAD(o.totalCents),
					icon: 'inbox' as const,
					status: { kind: 'order' as const, value: o.status },
					href: resolve(`/orders/${o.id}`)
				}))
			: []
	);

	const groups = $derived(
		[
			{ key: 'pages', heading: m.cmd_pages(), items: pageItems },
			{ key: 'actions', heading: m.cmd_actions(), items: actionItems },
			{ key: 'products', heading: m.cmd_products(), items: productItems },
			{ key: 'orders', heading: m.cmd_orders(), items: orderItems }
		].filter((g) => g.items.length > 0)
	);
	const fetching = $derived(searching && (products.isFetching || orders.isFetching));

	function select(item: Item) {
		open = false;
		// Hrefs are built with resolve() above.
		// eslint-disable-next-line svelte/no-navigation-without-resolve
		void goto(item.href);
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

<Dialog.Root
	bind:open
	onOpenChangeComplete={(isOpen) => {
		if (!isOpen) query = debounced = '';
	}}
>
	<Dialog.Portal>
		<Dialog.Overlay class="fixed inset-0 z-50 animate-fade-in bg-zinc-900/40 backdrop-blur-[2px]" />
		<Dialog.Content
			class="fixed top-[12vh] left-1/2 z-50 w-[calc(100%-2rem)] max-w-xl -translate-x-1/2 animate-pop-in overflow-hidden rounded-2xl bg-white text-zinc-900 shadow-popover focus:outline-none"
		>
			<Dialog.Title class="sr-only">{m.cmd_label()}</Dialog.Title>
			<Command.Root shouldFilter={false} loop label={m.cmd_label()}>
				<div class="flex items-center gap-3 border-b border-zinc-200 px-4">
					<Icon name="search" class="text-zinc-400" />
					<Command.Input
						bind:value={query}
						class="h-13 flex-1 bg-transparent text-[15px] placeholder:text-zinc-400 focus:outline-none"
						placeholder={m.cmd_placeholder()}
						maxlength={100}
						autocomplete="off"
						spellcheck={false}
					/>
					{#if fetching}
						<span
							class="size-4 animate-spin rounded-full border-2 border-zinc-200 border-t-brand-500"
							aria-hidden="true"
						></span>
					{/if}
					<kbd class="kbd">Esc</kbd>
				</div>
				<Command.List class="max-h-[50vh] overflow-y-auto p-2">
					<Command.Viewport>
						<Command.Empty class="px-3 py-8 text-center text-sm text-zinc-500">
							{fetching ? m.cmd_searching() : m.cmd_no_results()}
						</Command.Empty>
						{#each groups as group (group.key)}
							<Command.Group class="mb-1 last:mb-0">
								<Command.GroupHeading class="px-3 pt-2 pb-1 text-xs font-medium text-zinc-400">
									{group.heading}
								</Command.GroupHeading>
								<Command.GroupItems>
									{#each group.items as item (item.id)}
										<Command.Item
											value={item.id}
											keywords={[item.label]}
											onSelect={() => select(item)}
											class="flex cursor-pointer items-center gap-3 rounded-lg px-3 py-2 text-sm text-zinc-700 outline-none data-[selected]:bg-brand-100 data-[selected]:text-brand-900"
										>
											{#if item.status?.kind === 'product'}
												<Thumb src={item.image} class="size-7" rounded="rounded-md" iconSize={12} />
											{:else if item.icon}
												<span
													class="flex size-7 items-center justify-center rounded-md border border-zinc-200 bg-white text-zinc-500"
												>
													<Icon name={item.icon} size={15} />
												</span>
											{/if}
											<span class="min-w-0 flex-1 truncate font-medium">{item.label}</span>
											{#if item.status?.kind === 'product'}
												<StatusBadge kind="product" value={item.status.value} />
											{:else if item.status?.kind === 'order'}
												<StatusBadge kind="order" value={item.status.value as 'pending'} />
											{/if}
											{#if item.hint}<span class="text-xs text-zinc-500 tabular-nums"
													>{item.hint}</span
												>{/if}
										</Command.Item>
									{/each}
								</Command.GroupItems>
							</Command.Group>
						{/each}
					</Command.Viewport>
				</Command.List>
			</Command.Root>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>
