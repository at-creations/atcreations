<script lang="ts">
	import { serializeJsonLd } from '$lib/seo';

	interface Props {
		data: Record<string, unknown>;
	}

	let { data }: Props = $props();

	// serializeJsonLd escapes <, > and &, so the text below can't close the script element and
	// Svelte's text escaping (which only touches & and <) leaves the JSON untouched.
	const json = $derived(serializeJsonLd(data));
</script>

<svelte:head>
	<svelte:element this={"script"} type="application/ld+json">{json}</svelte:element>
</svelte:head>
