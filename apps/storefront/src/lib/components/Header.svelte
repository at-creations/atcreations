<script lang="ts">
	import { afterNavigate } from '$app/navigation';
	import { page } from '$app/state';
	import Search from '@lucide/svelte/icons/search';
	import Menu from '@lucide/svelte/icons/menu';
	import X from '@lucide/svelte/icons/x';
	import { m } from '$lib/paraglide/messages';
	import { localizedPath, stripLocale, type Locale } from '$lib/i18n';
	import { assetUrl } from '$lib/urls';
	import LocaleSwitcher from './LocaleSwitcher.svelte';

	interface Props {
		locale: Locale;
		assetBaseUrl: string;
	}

	let { locale, assetBaseUrl }: Props = $props();

	let menuOpen = $state(false);
	let menuButton: HTMLButtonElement | undefined = $state();
	const menuId = $props.id();

	const navItems = $derived([
		{ path: '/', label: m.nav_home() },
		{ path: '/products', label: m.nav_products() },
		{ path: '/workshops', label: m.nav_workshops() },
		{ path: '/about', label: m.nav_about() },
		{ path: '/contact', label: m.nav_contact() }
	]);

	const current = $derived(stripLocale(page.url.pathname));

	function isActive(path: string): boolean {
		return path === '/' ? current === '/' : current === path || current.startsWith(`${path}/`);
	}

	afterNavigate(() => {
		menuOpen = false;
	});

	function onKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape' && menuOpen) {
			menuOpen = false;
			menuButton?.focus();
		}
	}
</script>

<svelte:window onkeydown={onKeydown} />

<header class="fixed inset-x-0 top-0 z-40 bg-white shadow-md shadow-blue-900/10">
	<div class="container-page">
		<div class="flex h-16 items-center justify-between gap-4">
			<a href={localizedPath(locale)} class="flex shrink-0 items-center rounded">
				<img
					src={assetUrl(assetBaseUrl, 'logo-nav.png')}
					alt={m.nav_home_link()}
					width="96"
					height="32"
					class="h-8 w-auto"
				/>
			</a>

			<nav aria-label={m.nav_main_label()} class="hidden lg:block">
				<ul class="flex items-center gap-1">
					{#each navItems as item (item.path)}
						<li>
							<a
								href={localizedPath(locale, item.path)}
								aria-current={isActive(item.path) ? 'page' : undefined}
								class={[
									'block border-b-2 px-3 py-2 font-medium whitespace-nowrap transition-colors',
									isActive(item.path)
										? 'border-blue-600 text-blue-700'
										: 'border-transparent text-gray-700 hover:text-blue-700'
								]}
							>
								{item.label}
							</a>
						</li>
					{/each}
				</ul>
			</nav>

			<div class="flex items-center gap-2">
				<a
					href={localizedPath(locale, '/search')}
					class={[
						'rounded-full p-2 transition-colors hover:bg-gray-100',
						current.startsWith('/search') ? 'text-blue-700' : 'text-gray-700 hover:text-blue-700'
					]}
					aria-label={m.nav_search()}
					aria-current={current.startsWith('/search') ? 'page' : undefined}
				>
					<Search class="size-5" />
				</a>
				<LocaleSwitcher {locale} />
				<button
					bind:this={menuButton}
					type="button"
					class="rounded-full p-2 text-gray-700 hover:bg-gray-100 hover:text-blue-700 lg:hidden"
					aria-expanded={menuOpen}
					aria-controls={menuId}
					aria-label={menuOpen ? m.menu_close() : m.menu_open()}
					onclick={() => (menuOpen = !menuOpen)}
				>
					{#if menuOpen}<X class="size-6" />{:else}<Menu class="size-6" />{/if}
				</button>
			</div>
		</div>

		<nav
			id={menuId}
			aria-label={m.nav_main_label()}
			class={['border-t border-gray-200 py-3 lg:hidden', !menuOpen && 'hidden']}
		>
			<ul class="flex flex-col gap-1">
				{#each navItems as item (item.path)}
					<li>
						<a
							href={localizedPath(locale, item.path)}
							aria-current={isActive(item.path) ? 'page' : undefined}
							class={[
								'block rounded-md border-l-2 px-4 py-3 font-medium transition-colors',
								isActive(item.path)
									? 'border-blue-600 bg-gray-50 text-blue-700'
									: 'border-transparent text-gray-700 hover:bg-gray-50 hover:text-blue-700'
							]}
						>
							{item.label}
						</a>
					</li>
				{/each}
			</ul>
		</nav>
	</div>
</header>
