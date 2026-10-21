<script lang="ts">
	import type { Snippet } from 'svelte';
	import { m } from '$lib/paraglide/messages.js';

	type Lang = 'en' | 'vi';

	let {
		active = $bindable<Lang>('en'),
		errors = { en: false, vi: false },
		children
	}: {
		active?: Lang;
		errors?: { en: boolean; vi: boolean };
		/** Rendered once per language; only the active one is visible. */
		children: Snippet<[Lang]>;
	} = $props();

	const uid = $props.id();
	const langs: Lang[] = ['en', 'vi'];
	const labels = $derived({ en: m.tab_english(), vi: m.tab_vietnamese() });
	const tabs: Record<Lang, HTMLButtonElement | undefined> = $state({
		en: undefined,
		vi: undefined
	});

	function onkeydown(event: KeyboardEvent) {
		if (event.key !== 'ArrowLeft' && event.key !== 'ArrowRight') return;
		event.preventDefault();
		active = active === 'en' ? 'vi' : 'en';
		tabs[active]?.focus();
	}
</script>

<div>
	<div role="tablist" tabindex="-1" class="mb-4 flex gap-1 border-b border-slate-200" {onkeydown}>
		{#each langs as lang (lang)}
			<button
				bind:this={tabs[lang]}
				type="button"
				role="tab"
				id="{uid}-tab-{lang}"
				aria-controls="{uid}-panel-{lang}"
				aria-selected={active === lang}
				tabindex={active === lang ? 0 : -1}
				class={[
					'-mb-px border-b-2 px-4 py-2 text-sm font-medium',
					active === lang
						? 'border-brand-800 text-brand-800'
						: 'border-transparent text-slate-500 hover:text-slate-800',
					errors[lang] && 'text-red-600'
				]}
				onclick={() => (active = lang)}
			>
				{errors[lang] ? m.tab_has_errors({ tab: labels[lang] }) : labels[lang]}
			</button>
		{/each}
	</div>
	{#each langs as lang (lang)}
		<div
			role="tabpanel"
			id="{uid}-panel-{lang}"
			aria-labelledby="{uid}-tab-{lang}"
			hidden={active !== lang}
			class="space-y-4"
		>
			{@render children(lang)}
		</div>
	{/each}
</div>
