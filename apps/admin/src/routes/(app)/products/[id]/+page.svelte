<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { ApiError } from '$lib/api';
	import Icon from '$lib/components/Icon.svelte';
	import LoadState from '$lib/components/LoadState.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import ProductForm from '$lib/components/products/ProductForm.svelte';
	import ProductImages from '$lib/components/products/ProductImages.svelte';
	import StatusBadge from '$lib/components/StatusBadge.svelte';
	import { localized } from '$lib/i18n';
	import { storefrontProductUrl } from '$lib/links';
	import { m } from '$lib/paraglide/messages.js';
	import { categoriesQuery, productDetailQuery } from '$lib/queries';
	import { getAdminSession } from '$lib/state/session.svelte';

	const session = getAdminSession();
	const id = $derived(page.params.id ?? '');

	// The cache holds the server's latest product (and `version`): saves and image operations
	// write their responses into it, so the form and image manager always share one copy.
	const productQuery = createQuery(() => productDetailQuery(id));
	const categoriesList = createQuery(() => categoriesQuery());

	const product = $derived(productQuery.data);
	const notFound = $derived(
		productQuery.error instanceof ApiError &&
			(productQuery.error.status === 404 || productQuery.error.status === 400)
	);
	const loadError = $derived(
		(!product && productQuery.error) || (!categoriesList.data && categoriesList.error) || null
	);

	function retry() {
		if (productQuery.isError) void productQuery.refetch();
		if (categoriesList.isError) void categoriesList.refetch();
	}

	const siteUrl = $derived(
		product?.status === 'published' ? storefrontProductUrl(product.slug) : null
	);
</script>

<a
	href={resolve('/products')}
	class="mb-4 inline-flex items-center gap-1 text-sm text-slate-600 hover:underline"
>
	<Icon name="left" size={16} />{m.nav_products()}
</a>

{#if notFound}
	<div class="card p-10 text-center">
		<p class="font-medium">{m.pf_not_found()}</p>
		<a class="mt-4 btn-secondary" href={resolve('/products')}>{m.common_back()}</a>
	</div>
{:else if loadError}
	<LoadState error={loadError} onretry={retry} />
{:else if !product || !categoriesList.data}
	<LoadState loading />
{:else}
	<PageHeader
		title={session.can('products:write') ? m.pf_title_edit() : localized(product.name)}
		subtitle={localized(product.name)}
	>
		{#snippet actions()}
			<StatusBadge kind="product" value={product!.status} />
			{#if siteUrl}
				<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- external storefront URL -->
				<a class="btn-secondary" href={siteUrl} target="_blank" rel="noopener noreferrer">
					<Icon name="external" size={16} />{m.products_view_on_site()}
				</a>
			{/if}
		{/snippet}
	</PageHeader>
	<div class="grid gap-6">
		{#key product.id}
			<ProductForm {product} categories={categoriesList.data} />
		{/key}
		<ProductImages {product} />
	</div>
{/if}
