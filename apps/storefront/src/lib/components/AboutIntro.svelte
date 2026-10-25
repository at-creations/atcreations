<script lang="ts">
	import type { Snippet } from 'svelte';
	import { m } from '$lib/paraglide/messages';
	import { assetUrl } from '$lib/urls';

	interface Props {
		assetBaseUrl: string;
		/** Rendered above the paragraphs (e.g. a page heading). */
		heading?: Snippet;
	}

	let { assetBaseUrl, heading }: Props = $props();
</script>

<div class="grid grid-cols-1 items-end gap-12 lg:grid-cols-12">
	<div class="lg:col-span-7">
		{@render heading?.()}
		<div class="max-w-2xl space-y-5 text-lg leading-relaxed">
			<p>{m.about_paragraph1()}</p>
			<p>{m.about_paragraph2()}</p>
			<p>{m.about_paragraph3()}</p>
		</div>
		<blockquote class="mt-8 max-w-xl font-display text-2xl leading-snug font-semibold text-ink">
			{m.about_quote()}
		</blockquote>
		<p class="mt-3 text-ink-soft">{m.about_established()}</p>
	</div>

	<div class="mx-auto w-full max-w-md lg:col-span-5 lg:max-w-none">
		<img
			src={assetUrl(assetBaseUrl, 'about_row.jpg')}
			alt={m.about_image_alt()}
			width="1276"
			height="956"
			class="aspect-[4/3] w-full rounded-t-lg object-cover"
		/>
		<div class="shelf [--shelf-gap:2.5rem]" aria-hidden="true"></div>
	</div>
</div>
