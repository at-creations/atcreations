<script lang="ts">
	import type { Permission } from '@atcreations/api-types';
	import type { Pathname } from '$app/types';
	import { UserButton } from 'svelte-clerk';
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import { m } from '$lib/paraglide/messages.js';
	import { getAdminSession } from '$lib/state/session.svelte';
	import Icon, { type IconName } from './Icon.svelte';
	import LocaleSwitcher from './LocaleSwitcher.svelte';
	import StatusBadge from './StatusBadge.svelte';

	let { onnavigate }: { onnavigate?: () => void } = $props();

	const session = getAdminSession();

	type Href = Pathname;
	interface NavItem {
		href: Href;
		label: () => string;
		icon: IconName;
		permission?: Permission;
	}

	const items: NavItem[] = [
		{ href: '/', label: m.nav_dashboard, icon: 'dashboard' },
		{ href: '/products', label: m.nav_products, icon: 'box' },
		{ href: '/categories', label: m.nav_categories, icon: 'tag' },
		{ href: '/orders', label: m.nav_orders, icon: 'cart', permission: 'orders:read' },
		{ href: '/staff', label: m.nav_staff, icon: 'users', permission: 'staff:manage' },
		{ href: '/store', label: m.nav_store, icon: 'store', permission: 'store:write' },
		{ href: '/audit', label: m.nav_audit, icon: 'history', permission: 'audit:read' }
	];

	const visible = $derived(items.filter((i) => !i.permission || session.can(i.permission)));

	function isActive(href: Href) {
		const path: string = page.url.pathname;
		return href === '/' ? path === '/' : path === href || path.startsWith(`${href}/`);
	}
</script>

<div class="flex h-full flex-col bg-[#172554] text-white">
	<div class="flex items-center gap-2 px-5 py-5">
		<a href={resolve('/')} class="text-lg font-bold tracking-tight" onclick={onnavigate}
			>{m.app_name()}</a
		>
		<span
			class="rounded bg-white/15 px-1.5 py-0.5 text-[10px] font-semibold tracking-wider uppercase"
			>{m.admin_badge()}</span
		>
	</div>
	<nav aria-label={m.nav_main()} class="flex-1 overflow-y-auto px-3">
		<ul class="space-y-1">
			{#each visible as item (item.href)}
				{@const active = isActive(item.href)}
				<li>
					<a
						href={resolve(item.href)}
						aria-current={active ? 'page' : undefined}
						class={[
							'flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition',
							active ? 'bg-white/15 text-white' : 'text-blue-100 hover:bg-white/10 hover:text-white'
						]}
						onclick={onnavigate}
					>
						<Icon name={item.icon} />
						{item.label()}
					</a>
				</li>
			{/each}
		</ul>
	</nav>
	<div class="space-y-4 border-t border-white/10 px-5 py-4">
		<LocaleSwitcher />
		<div class="flex items-center gap-3">
			<UserButton />
			{#if session.me}
				<div class="min-w-0 text-sm">
					<p class="truncate font-medium">{session.me.name || session.me.email}</p>
					<StatusBadge kind="role" value={session.me.role} />
				</div>
			{/if}
		</div>
	</div>
</div>
