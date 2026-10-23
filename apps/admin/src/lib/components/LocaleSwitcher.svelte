<script lang="ts">
	import { Select } from 'bits-ui';
	import { m } from '$lib/paraglide/messages.js';
	import { getLocale, locales, setLocale } from '$lib/paraglide/runtime.js';
	import Icon from './Icon.svelte';

	let { compact = false }: { /** Show only the language code (top bar). */ compact?: boolean } =
		$props();

	type Locale = (typeof locales)[number];
	const labels: Record<Locale, () => string> = { en: m.lang_en, vi: m.lang_vi };
	const items = $derived(locales.map((l) => ({ value: l, label: labels[l]() })));
	const current = getLocale();
</script>

<Select.Root
	type="single"
	value={current}
	{items}
	onValueChange={(v) => {
		if (v && v !== current) setLocale(v as Locale);
	}}
>
	<Select.Trigger
		aria-label={m.language()}
		title={m.language()}
		class="flex h-9 items-center gap-1.5 rounded-lg px-2.5 text-sm font-medium text-zinc-600 transition-colors hover:bg-zinc-100 hover:text-zinc-900 focus-visible:ring-2 focus-visible:ring-brand-300 focus-visible:outline-none data-[state=open]:bg-zinc-100"
	>
		<Icon name="globe" size={16} />
		<span class={compact ? 'uppercase' : ''}>{compact ? current : labels[current]()}</span>
	</Select.Trigger>
	<Select.Portal>
		<Select.Content
			sideOffset={4}
			align="end"
			class="z-[60] min-w-36 animate-pop-in rounded-xl border border-zinc-200 bg-white p-1 shadow-popover"
		>
			<Select.Viewport>
				{#each items as item (item.value)}
					<Select.Item
						value={item.value}
						label={item.label}
						class="flex h-8 cursor-pointer items-center justify-between gap-3 rounded-lg px-2.5 text-sm text-zinc-700 outline-none select-none data-[highlighted]:bg-brand-50 data-[highlighted]:text-brand-900"
					>
						{#snippet children({ selected })}
							{item.label}
							{#if selected}<Icon name="check" size={14} class="text-brand-600" />{/if}
						{/snippet}
					</Select.Item>
				{/each}
			</Select.Viewport>
		</Select.Content>
	</Select.Portal>
</Select.Root>
