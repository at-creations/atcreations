<script lang="ts">
	import { m } from '$lib/paraglide/messages.js';
	import { getLocale, locales, setLocale } from '$lib/paraglide/runtime.js';
	import Icon from './Icon.svelte';

	let { compact = false }: { /** Show only the language code (top bar). */ compact?: boolean } =
		$props();

	const uid = $props.id();
	const labels: Record<(typeof locales)[number], () => string> = {
		en: m.lang_en,
		vi: m.lang_vi
	};
</script>

<div
	class="relative flex h-9 items-center rounded-lg text-zinc-600 focus-within:ring-2 focus-within:ring-brand-600/30 hover:bg-zinc-100 hover:text-zinc-900"
>
	<Icon name="globe" size={16} class="pointer-events-none absolute left-2.5" />
	<label class="sr-only" for="{uid}-locale">{m.language()}</label>
	<select
		id="{uid}-locale"
		class={[
			'h-full cursor-pointer appearance-none rounded-lg bg-transparent pr-3 pl-8 text-sm font-medium focus:outline-none',
			compact && 'uppercase'
		]}
		title={m.language()}
		value={getLocale()}
		onchange={(event) => setLocale(event.currentTarget.value as (typeof locales)[number])}
	>
		{#each locales as locale (locale)}
			<option value={locale}>{compact ? locale.toUpperCase() : labels[locale]()}</option>
		{/each}
	</select>
</div>
