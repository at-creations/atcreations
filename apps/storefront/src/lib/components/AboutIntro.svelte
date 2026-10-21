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

<div class="grid grid-cols-1 items-center gap-12 lg:grid-cols-12">
	<div class="order-2 lg:order-1 lg:col-span-7">
		{@render heading?.()}
		<div class="relative space-y-6 text-lg leading-relaxed text-gray-700">
			<div
				class="absolute top-0 -left-6 hidden h-24 w-1 rounded-full bg-blue-400 lg:block"
				aria-hidden="true"
			></div>
			<p>{m.about_paragraph1()}</p>
			<p>{m.about_paragraph2()}</p>
			<p>{m.about_paragraph3()}</p>
			<blockquote class="border-l-4 border-blue-400 pl-4 text-gray-600 italic">
				{m.about_quote()}
			</blockquote>
		</div>
	</div>

	<div class="order-1 lg:order-2 lg:col-span-5">
		<div class="relative">
			<div
				class="absolute -right-4 -bottom-4 size-full rounded-xl bg-blue-100"
				aria-hidden="true"
			></div>
			<picture class="relative z-10 block overflow-hidden rounded-xl shadow-xl">
				<source
					media="(min-width: 1024px)"
					srcset={assetUrl(assetBaseUrl, 'about_col.jpg')}
					width="765"
					height="956"
				/>
				<img
					src={assetUrl(assetBaseUrl, 'about_row.jpg')}
					alt={m.about_image_alt()}
					width="1276"
					height="956"
					loading="lazy"
					class="h-auto w-full object-cover lg:aspect-[4/5]"
				/>
			</picture>
			<p
				class="absolute -bottom-6 -left-6 z-20 hidden rounded-full bg-white px-4 py-2 font-medium text-blue-700 shadow-lg lg:block"
			>
				{m.about_established()}
			</p>
		</div>
	</div>
</div>
