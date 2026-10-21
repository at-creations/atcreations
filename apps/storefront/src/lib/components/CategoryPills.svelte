<script lang="ts">
	import type { Category } from '@atcreations/api-types';
	import { m } from '$lib/paraglide/messages';
	import { t, type Locale } from '$lib/i18n';
	import { productsHref } from '$lib/urls';

	interface Props {
		categories: Category[];
		/** Selected category slug, or null for "All". */
		selected: string | null;
		locale: Locale;
	}

	let { categories, selected, locale }: Props = $props();

	const pill =
		'inline-block rounded-full px-4 py-2 text-sm font-medium transition-colors duration-200';
	const active = 'bg-blue-600 text-white shadow-md hover:bg-blue-700';
	const idle =
		'border border-gray-200 bg-white text-gray-700 hover:border-blue-500 hover:text-blue-700';
</script>

<nav aria-label={m.products_categories()}>
	<ul class="flex flex-wrap justify-center gap-2">
		<li>
			<a
				href={productsHref(locale)}
				class={[pill, selected === null ? active : idle]}
				aria-current={selected === null ? 'page' : undefined}
				data-sveltekit-noscroll
			>
				{m.products_all()}
			</a>
		</li>
		{#each categories as category (category.id)}
			<li>
				<a
					href={productsHref(locale, { category: category.slug })}
					class={[pill, selected === category.slug ? active : idle]}
					aria-current={selected === category.slug ? 'page' : undefined}
					data-sveltekit-noscroll
				>
					{t(category.name, locale)}
				</a>
			</li>
		{/each}
	</ul>
</nav>
