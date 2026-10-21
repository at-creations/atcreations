<script lang="ts">
	import { m } from '$lib/paraglide/messages.js';
	import { getLocale, locales, setLocale } from '$lib/paraglide/runtime.js';
	import Icon from './Icon.svelte';

	const uid = $props.id();
	const labels: Record<(typeof locales)[number], () => string> = {
		en: m.lang_en,
		vi: m.lang_vi
	};
</script>

<div class="flex items-center gap-2">
	<Icon name="globe" size={16} class="text-blue-200" />
	<label class="sr-only" for="{uid}-locale">{m.language()}</label>
	<select
		id="{uid}-locale"
		class="w-full rounded-md border border-white/20 bg-white/10 px-2 py-1 text-sm text-white focus:ring-2 focus:ring-white/40 focus:outline-none"
		value={getLocale()}
		onchange={(event) => setLocale(event.currentTarget.value as (typeof locales)[number])}
	>
		{#each locales as locale (locale)}
			<option value={locale} class="text-slate-900">{labels[locale]()}</option>
		{/each}
	</select>
</div>
