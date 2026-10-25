<script lang="ts">
	import { page } from '$app/state';
	import ChevronDown from '@lucide/svelte/icons/chevron-down';
	import Check from '@lucide/svelte/icons/check';
	import { m } from '$lib/paraglide/messages';
	import { LOCALES, switchLocaleHref, type Locale } from '$lib/i18n';

	interface Props {
		locale: Locale;
	}

	let { locale }: Props = $props();

	let open = $state(false);
	let root: HTMLDivElement | undefined = $state();
	const menuId = $props.id();

	const LABELS: Record<Locale, () => string> = {
		en: m.language_en,
		vi: m.language_vi
	};
	const FLAGS: Record<Locale, string> = { en: '/icons/ca.svg', vi: '/icons/vi.svg' };

	function remember(target: Locale) {
		// Non-sensitive preference, read by the server to pick the locale for `/`.
		document.cookie = `PARAGLIDE_LOCALE=${target}; Path=/; Max-Age=34560000; SameSite=Lax`;
		open = false;
	}

	function onWindowClick(event: MouseEvent) {
		if (open && root && !root.contains(event.target as Node)) open = false;
	}

	function onKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape' && open) {
			open = false;
			root?.querySelector('button')?.focus();
		}
	}
</script>

<svelte:window onclick={onWindowClick} />

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="relative" bind:this={root} onkeydown={onKeydown}>
	<button
		type="button"
		class="flex items-center gap-1.5 rounded-full px-2 py-1.5 text-sm font-medium text-ink-soft hover:bg-wall hover:text-ink"
		aria-expanded={open}
		aria-controls={menuId}
		aria-label={m.language_current({ language: LABELS[locale]() })}
		onclick={() => (open = !open)}
	>
		<img
			src={FLAGS[locale]}
			alt=""
			width="20"
			height="20"
			class="size-5 rounded-full object-cover"
		/>
		<span class="uppercase">{locale}</span>
		<ChevronDown class={['size-4 transition-transform', open && 'rotate-180']} />
	</button>

	<ul
		id={menuId}
		class={[
			'absolute right-0 z-50 mt-2 w-44 overflow-hidden rounded-xl border border-line bg-white py-1 shadow-lg',
			!open && 'hidden'
		]}
	>
		{#each LOCALES as target (target)}
			<li>
				<a
					href={switchLocaleHref(page.url, target)}
					hreflang={target}
					lang={target}
					data-sveltekit-reload
					aria-current={target === locale ? 'true' : undefined}
					class="flex items-center gap-2 px-3 py-2 text-sm text-ink-soft hover:bg-mist hover:text-ink"
					onclick={() => remember(target)}
				>
					<img
						src={FLAGS[target]}
						alt=""
						width="20"
						height="20"
						class="size-5 rounded-full object-cover"
					/>
					<span class="flex-1">{LABELS[target]()}</span>
					{#if target === locale}<Check class="size-4 text-ink" />{/if}
				</a>
			</li>
		{/each}
	</ul>
</div>
