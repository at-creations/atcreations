<script lang="ts">
	import { srcset } from '$lib/images';
	import Clock from '@lucide/svelte/icons/clock';
	import Users from '@lucide/svelte/icons/users';
	import { m } from '$lib/paraglide/messages';
	import { localizedPath, t } from '$lib/i18n';
	import { assetUrl, productsHref } from '$lib/urls';
	import Seo from '$lib/components/Seo.svelte';
	import Hero from '$lib/components/Hero.svelte';
	import SectionTitle from '$lib/components/SectionTitle.svelte';
	import ProductCard from '$lib/components/ProductCard.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const locale = $derived(data.locale);
	const asset = (file: string) => assetUrl(data.config.assetBaseUrl, file);
</script>

<Seo title={m.home_meta_title()} rawTitle description={m.site_description()} />

<Hero {locale} assetBaseUrl={data.config.assetBaseUrl} storeInfo={data.storeInfo} />

{#if data.categories.length > 0}
	<section class="container-page pt-20 md:pt-24" aria-labelledby="categories-title">
		<SectionTitle id="categories-title" title={m.categories_title()} />
		<ul class="grid grid-cols-2 gap-x-6 gap-y-10 sm:grid-cols-3 lg:grid-cols-5">
			{#each data.categories as category (category.id)}
				<li>
					<a href={productsHref(locale, { category: category.slug })} class="group block">
						<div class="aspect-square overflow-hidden rounded-t-md bg-wall">
							{#if category.image}
								<img
									src={category.image.url}
									srcset={srcset(category.image)}
									sizes="(min-width: 1024px) 20vw, (min-width: 640px) 33vw, 50vw"
									alt=""
									width={category.image.width}
									height={category.image.height}
									loading="lazy"
									class="size-full object-cover transition-transform duration-500 group-hover:scale-[1.03] motion-reduce:group-hover:scale-100"
								/>
							{/if}
						</div>
						<div class="shelf" aria-hidden="true"></div>
						<p
							class="mt-3 font-display text-lg font-semibold text-ink group-hover:underline group-hover:decoration-marigold group-hover:decoration-2 group-hover:underline-offset-4"
						>
							{t(category.name, locale)}
						</p>
					</a>
				</li>
			{/each}
		</ul>
	</section>
{/if}

{#if data.featured.length > 0}
	<section class="container-page pt-20 md:pt-24" aria-labelledby="featured-title">
		<SectionTitle id="featured-title" title={m.featured_title()}>
			{#snippet action()}
				<a href={productsHref(locale)} class="link">{m.featured_view_all()}</a>
			{/snippet}
		</SectionTitle>
		<ul class="grid grid-cols-2 gap-x-6 gap-y-12 lg:grid-cols-4">
			{#each data.featured as product (product.id)}
				<li><ProductCard {product} {locale} /></li>
			{/each}
		</ul>
	</section>
{/if}

<section class="mt-24 bg-mist md:mt-28" aria-labelledby="workshop-title">
	<div class="container-page grid items-center gap-10 py-16 md:grid-cols-12 md:gap-16 md:py-20">
		<img
			src={asset('plaster-models.jpg')}
			alt={m.plaster_img_models()}
			width="1536"
			height="2048"
			loading="lazy"
			class="aspect-[4/5] w-full max-w-sm rounded-lg object-cover md:col-span-5 md:max-w-none lg:col-span-4"
		/>
		<div class="md:col-span-7 lg:col-span-6">
			<h2 id="workshop-title" class="heading-section md:text-4xl">{m.home_workshop_title()}</h2>
			<p class="mt-5 max-w-lg text-lg leading-relaxed">{m.home_workshop_text()}</p>
			<ul class="mt-7 flex flex-wrap gap-x-8 gap-y-3 font-medium text-ink">
				<li class="flex items-center gap-2">
					<Clock class="size-5 text-teak" />{m.workshops_duration()}
				</li>
				<li class="flex items-center gap-2">
					<Users class="size-5 text-teak" />{m.workshops_capacity()}
				</li>
			</ul>
			<a href={localizedPath(locale, '/workshops/plaster-painting')} class="mt-9 btn-primary">
				{m.home_workshop_cta()}
			</a>
		</div>
	</div>
</section>

<section class="container-page py-20 md:py-28" aria-labelledby="story-title">
	<div class="grid items-center gap-10 md:grid-cols-12 md:gap-16">
		<div class="md:col-span-6 lg:col-span-5">
			<h2 id="story-title" class="heading-section md:text-4xl">{m.home_story_title()}</h2>
			<p class="mt-5 max-w-lg text-lg leading-relaxed">{m.home_story_text()}</p>
			<a href={localizedPath(locale, '/about')} class="mt-8 inline-block link">{m.about_more()}</a>
		</div>
		<div class="md:col-span-6 lg:col-span-7">
			<img
				src={asset('about_row.jpg')}
				alt={m.about_image_alt()}
				width="1276"
				height="956"
				loading="lazy"
				class="aspect-[4/3] w-full rounded-lg object-cover"
			/>
		</div>
	</div>
</section>
