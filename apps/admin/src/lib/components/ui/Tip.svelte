<script lang="ts">
	import { Tooltip } from 'bits-ui';
	import type { Snippet } from 'svelte';

	let {
		text,
		side = 'top',
		children
	}: {
		text: string;
		side?: 'top' | 'bottom' | 'left' | 'right';
		/** The trigger; spread `props` onto the focusable element (button or link). */
		children: Snippet<[{ props: Record<string, unknown> }]>;
	} = $props();
</script>

<Tooltip.Root>
	<Tooltip.Trigger>
		{#snippet child({ props })}
			{@render children({ props })}
		{/snippet}
	</Tooltip.Trigger>
	<Tooltip.Portal>
		<Tooltip.Content
			{side}
			sideOffset={6}
			class="z-[70] animate-pop-in rounded-md bg-zinc-900 px-2 py-1 text-xs font-medium text-white shadow-lg"
		>
			{text}
		</Tooltip.Content>
	</Tooltip.Portal>
</Tooltip.Root>
