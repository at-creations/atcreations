<script lang="ts">
	interface Props {
		title: string;
		/** Heading level (1 for page titles, 2 for sections). */
		level?: 1 | 2;
		centered?: boolean;
		id?: string;
		class?: string;
	}

	let { title, level = 2, centered = true, id, class: className = '' }: Props = $props();
</script>

<div class={['relative mb-14', centered && 'text-center', className]}>
	<svelte:element
		this={level === 1 ? 'h1' : 'h2'}
		{id}
		class="inline-block text-3xl font-bold tracking-tight text-gray-900 md:text-4xl"
	>
		{title}
	</svelte:element>
	<div
		class={[
			'absolute -bottom-3 h-1 rounded-full bg-blue-500',
			centered ? 'left-1/2 w-24 -translate-x-1/2' : 'left-0 w-20'
		]}
		aria-hidden="true"
	></div>
</div>
