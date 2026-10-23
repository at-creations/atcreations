<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { isActivePath, NAV } from '$lib/nav';
	import { m } from '$lib/paraglide/messages.js';
	import { pendingOrderCountQuery } from '$lib/queries';
	import { getAdminSession } from '$lib/state/session.svelte';
	import Icon from './Icon.svelte';

	let { onnavigate }: { onnavigate?: () => void } = $props();

	const session = getAdminSession();

	const sections = $derived(
		NAV.map((items) => items.filter((i) => !i.permission || session.can(i.permission))).filter(
			(items) => items.length > 0
		)
	);

	const pending = createQuery(() => ({
		...pendingOrderCountQuery(),
		enabled: session.can('orders:read')
	}));
</script>

<div class="flex h-full flex-col border-r border-zinc-200 bg-white">
	<a
		href={resolve('/')}
		class="flex h-14 shrink-0 items-center gap-2.5 px-5 focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-brand-600"
		onclick={onnavigate}
	>
		<img src="/apple-touch-icon.png" alt="" class="size-7 rounded-md" />
		<span class="text-sm font-semibold text-zinc-900">{m.app_name()}</span>
	</a>

	<nav aria-label={m.nav_main()} class="flex-1 overflow-y-auto px-3 py-2">
		{#each sections as items, si (si)}
			{#if si > 0}<hr class="mx-2 my-3 border-zinc-100" />{/if}
			<ul class="space-y-0.5">
				{#each items as item (item.href)}
					{@const active = isActivePath(item.href, page.url.pathname)}
					{@const count = item.badge === 'pendingOrders' ? (pending.data ?? 0) : 0}
					<li>
						<a
							href={resolve(item.href)}
							aria-current={active ? 'page' : undefined}
							class={[
								'flex h-9 items-center gap-3 rounded-lg px-2.5 text-sm font-medium transition-colors',
								active
									? 'bg-brand-100 text-brand-900'
									: 'text-zinc-600 hover:bg-zinc-50 hover:text-zinc-900'
							]}
							onclick={onnavigate}
						>
							<Icon name={item.icon} class={active ? 'text-brand-700' : 'text-zinc-400'} />
							<span class="flex-1">{item.label()}</span>
							{#if count > 0}
								<span
									class="rounded-full bg-amber-100 px-1.5 text-[11px] leading-5 font-semibold text-amber-800 tabular-nums"
									>{count > 99 ? '99+' : count}<span class="sr-only">
										{m.nav_pending_count({ count })}</span
									></span
								>
							{/if}
						</a>
					</li>
				{/each}
			</ul>
		{/each}
	</nav>
</div>
