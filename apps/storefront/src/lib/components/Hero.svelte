<script lang="ts">
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import { m } from '$lib/paraglide/messages';
	import { localizedPath, type Locale } from '$lib/i18n';
	import { assetUrl } from '$lib/urls';

	interface Props {
		locale: Locale;
		assetBaseUrl: string;
	}

	let { locale, assetBaseUrl }: Props = $props();
</script>

<section
	class="relative flex min-h-svh flex-col items-center justify-center overflow-hidden bg-gray-900"
	aria-labelledby="hero-title"
>
	<img
		src={assetUrl(assetBaseUrl, 'hero.jpg')}
		alt=""
		class="absolute inset-0 size-full scale-105 object-cover"
		fetchpriority="high"
	/>
	<div class="absolute inset-0 bg-black/50" aria-hidden="true"></div>
	<div
		class="absolute inset-0 bg-linear-to-br from-blue-900/40 via-transparent to-black/70"
		aria-hidden="true"
	></div>

	<div class="relative z-10 mx-auto flex max-w-5xl flex-col items-center px-6 text-center">
		<h1 id="hero-title" class="animate-fade-in">
			<img
				src={assetUrl(assetBaseUrl, 'logo-hero.png')}
				alt={m.home_logo_alt()}
				width="1200"
				height="435"
				class="h-36 w-auto sm:h-52 md:h-64"
			/>
		</h1>
		<p
			class="mt-8 mb-10 max-w-2xl animate-slide-up text-lg font-light tracking-wide text-white drop-shadow-md md:text-xl"
		>
			{m.home_tagline()}
		</p>
		<a
			href={localizedPath(locale, '/products')}
			class="group inline-flex animate-slide-up items-center gap-2 rounded-full border border-white/40 bg-white/20 px-8 py-4 font-medium text-white shadow-xl backdrop-blur-md transition hover:bg-white/30 focus-visible:outline-white"
		>
			{m.home_explore()}
			<ArrowRight class="size-5 transition-transform group-hover:translate-x-1" />
		</a>
	</div>
</section>
