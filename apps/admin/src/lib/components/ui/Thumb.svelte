<script lang="ts">
	import { safeImageUrl } from '$lib/links';
	import Icon from '../Icon.svelte';

	let {
		src,
		alt = '',
		class: className = 'size-10',
		rounded = 'rounded-lg',
		iconSize = 16
	}: {
		/** Untrusted URL from the API; only http(s) is rendered. */
		src?: string | null;
		alt?: string;
		/** Size (and any extra) classes. */
		class?: string;
		rounded?: string;
		iconSize?: number;
	} = $props();

	const url = $derived(safeImageUrl(src));
	let failed = $state(false);
	$effect(() => {
		void url;
		failed = false;
	});
</script>

{#if url && !failed}
	<img
		src={url}
		{alt}
		loading="lazy"
		decoding="async"
		class={['shrink-0 bg-zinc-100 object-cover', rounded, className]}
		onerror={() => (failed = true)}
	/>
{:else}
	<span
		class={[
			'flex shrink-0 items-center justify-center bg-zinc-100 text-zinc-300',
			rounded,
			className
		]}
		role={alt ? 'img' : undefined}
		aria-label={alt || undefined}
	>
		<Icon name="image" size={iconSize} />
	</span>
{/if}
