<script lang="ts">
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import { m } from '$lib/paraglide/messages';
	import { localizedPath, t } from '$lib/i18n';
	import { productsHref } from '$lib/urls';
	import Seo from '$lib/components/Seo.svelte';
	import Hero from '$lib/components/Hero.svelte';
	import SectionTitle from '$lib/components/SectionTitle.svelte';
	import ProductCard from '$lib/components/ProductCard.svelte';
	import AboutIntro from '$lib/components/AboutIntro.svelte';
	import ContactDetails from '$lib/components/ContactDetails.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const locale = $derived(data.locale);
</script>

<Seo title={m.home_meta_title()} rawTitle description={m.site_description()} />

<Hero {locale} assetBaseUrl={data.config.assetBaseUrl} />

<div class="space-y-24 py-20">
	{#if data.featured.length > 0}
		<section class="container-page animate-fade-in" aria-labelledby="featured-title">
			<SectionTitle id="featured-title" title={m.featured_title()} />
			<ul class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
				{#each data.featured as product (product.id)}
					<li><ProductCard {product} {locale} /></li>
				{/each}
			</ul>
			<div class="mt-12 flex justify-center">
				<a href={productsHref(locale)} class="group btn-primary">
					{m.featured_view_all()}
					<ArrowRight class="size-5 transition-transform group-hover:translate-x-1" />
				</a>
			</div>
		</section>
	{/if}

	{#if data.categories.length > 0}
		<section class="container-page" aria-labelledby="categories-title">
			<SectionTitle id="categories-title" title={m.categories_title()} />
			<p class="mx-auto -mt-4 mb-10 max-w-2xl text-center text-gray-600">
				{m.categories_description()}
			</p>
			<ul class="mx-auto grid max-w-3xl grid-cols-2 gap-4 sm:grid-cols-3 md:gap-6">
				{#each data.categories as category (category.id)}
					<li>
						<a
							href={productsHref(locale, { category: category.slug })}
							class="group relative block aspect-square overflow-hidden rounded-xl shadow-card transition duration-200 hover:-translate-y-1 hover:shadow-card-hover motion-reduce:hover:translate-y-0"
						>
							{#if category.image}
								<img
									src={category.image.url}
									alt=""
									width={category.image.width}
									height={category.image.height}
									loading="lazy"
									class="absolute inset-0 size-full object-cover transition-transform duration-500 group-hover:scale-105"
								/>
							{:else}
								<div class="absolute inset-0 bg-linear-to-br from-blue-100 to-blue-300"></div>
							{/if}
							<div
								class="absolute inset-0 bg-linear-to-t from-black/75 via-black/30 to-transparent"
								aria-hidden="true"
							></div>
							<span
								class="absolute inset-x-0 bottom-0 truncate p-3 font-medium text-white sm:text-base"
							>
								{t(category.name, locale)}
							</span>
						</a>
					</li>
				{/each}
			</ul>
		</section>
	{/if}

	<section class="container-page max-w-6xl" aria-labelledby="about-title">
		<SectionTitle id="about-title" title={m.about_title()} />
		<AboutIntro assetBaseUrl={data.config.assetBaseUrl} />
		<div class="mt-16 flex justify-center">
			<a href={localizedPath(locale, '/about')} class="group btn-primary">
				{m.about_more()}
				<ArrowRight class="size-5 transition-transform group-hover:translate-x-1" />
			</a>
		</div>
	</section>

	<section class="container-page" aria-labelledby="contact-title">
		<SectionTitle id="contact-title" title={m.contact_title()} />
		<ContactDetails storeInfo={data.storeInfo} {locale} />
	</section>
</div>
