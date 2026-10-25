<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		title: string;
		/** Heading level (1 for page titles, 2 for sections). */
		level?: 1 | 2;
		centered?: boolean;
		id?: string;
		class?: string;
		/** Shown at the end of the title row (e.g. a "See all" link). */
		action?: Snippet;
	}

	let { title, level = 2, centered = false, id, class: className = '', action }: Props = $props();
</script>

<div
	class={[
		'mb-8 flex flex-wrap items-end gap-x-6 gap-y-3 md:mb-10',
		centered ? 'justify-center text-center' : 'justify-between',
		className
	]}
>
	<svelte:element
		this={level === 1 ? 'h1' : 'h2'}
		{id}
		class={level === 1 ? 'heading-page' : 'heading-section'}
	>
		{title}
	</svelte:element>
	{@render action?.()}
</div>
