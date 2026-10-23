<script lang="ts">
	import { Tabs } from 'bits-ui';
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

	const langs: Lang[] = ['en', 'vi'];
	const labels = $derived({ en: m.tab_english(), vi: m.tab_vietnamese() });
</script>

<Tabs.Root bind:value={active as string}>
	<Tabs.List class="mb-4 flex gap-1 border-b border-zinc-200">
		{#each langs as lang (lang)}
			<Tabs.Trigger
				value={lang}
				class={[
					'-mb-px border-b-2 border-transparent px-4 py-2 text-sm font-medium text-zinc-500 transition-colors hover:text-zinc-800 focus-visible:bg-zinc-50 focus-visible:outline-none data-[state=active]:border-brand-400 data-[state=active]:text-brand-800',
					errors[lang] && 'text-red-600 data-[state=active]:text-red-600'
				]}
			>
				{errors[lang] ? m.tab_has_errors({ tab: labels[lang] }) : labels[lang]}
			</Tabs.Trigger>
		{/each}
	</Tabs.List>
	{#each langs as lang (lang)}
		<!-- Field values live in the parent form, so switching tabs never loses input. -->
		<Tabs.Content value={lang} class="space-y-4">
			{@render children(lang)}
		</Tabs.Content>
	{/each}
</Tabs.Root>
