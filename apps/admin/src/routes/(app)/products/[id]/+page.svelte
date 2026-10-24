<script lang="ts">
	import { imageUrl } from '$lib/utils/images';
	import { createQuery } from '@tanstack/svelte-query';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { ApiError } from '$lib/api';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import Icon from '$lib/components/Icon.svelte';
	import LoadState from '$lib/components/LoadState.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import ProductForm from '$lib/components/products/ProductForm.svelte';
	import ProductImages from '$lib/components/products/ProductImages.svelte';
	import StatusBadge from '$lib/components/StatusBadge.svelte';
	import { localized } from '$lib/i18n';
	import { storefrontProductUrl } from '$lib/links';
	import { m } from '$lib/paraglide/messages.js';
	import Thumb from '$lib/components/ui/Thumb.svelte';
	import { toastError } from '$lib/notify';
	import { categoriesQuery, createDeleteProductMutation, productDetailQuery } from '$lib/queries';
	import { confirmAction } from '$lib/state/confirm.svelte';
	import { getAdminSession } from '$lib/state/session.svelte';
	import { toast } from '$lib/state/toast.svelte';
	import { goto } from '$app/navigation';

	const session = getAdminSession();
	const deleteProduct = createDeleteProductMutation();
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

	async function remove() {
		if (!product) return;
		const ok = await confirmAction({
			title: m.products_delete_title(),
			message: m.products_delete_message({ name: localized(product.name) }),
			confirmLabel: m.common_delete(),
			danger: true
		});
		if (!ok) return;
		try {
			await deleteProduct.mutateAsync(product.id);
			toast.success(m.products_deleted());
			await goto(resolve('/products'));
		} catch (error) {
			toastError(error, m.products_delete_failed());
		}
	}

	const siteUrl = $derived(
		product?.status === 'published' ? storefrontProductUrl(product.slug) : null
	);
</script>

{#if notFound}
	<div class="card">
		<EmptyState icon="box" title={m.pf_not_found()}>
			<a class="btn-secondary" href={resolve('/products')}
				><Icon name="left" size={16} />{m.nav_products()}</a
			>
		</EmptyState>
	</div>
{:else if loadError}
	<div class="card"><LoadState error={loadError} onretry={retry} /></div>
{:else if !product || !categoriesList.data}
	<div class="space-y-6" role="status" aria-label={m.common_loading()}>
		<div class="h-8 w-64 animate-pulse rounded-md bg-zinc-200"></div>
		<div class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_20rem]">
			<div class="h-96 animate-pulse card"></div>
			<div class="h-64 animate-pulse card"></div>
		</div>
	</div>
{:else}
	<PageHeader
		title={localized(product.name)}
		back={{ href: resolve('/products'), label: m.nav_products() }}
	>
		{#snippet leading()}
			<Thumb
				src={imageUrl(product!.images[0], 96)}
				class="hidden size-12 sm:flex"
				rounded="rounded-xl"
			/>
		{/snippet}
		{#snippet meta()}
			<StatusBadge kind="product" value={product!.status} />
		{/snippet}
		{#snippet actions()}
			{#if session.can('products:delete')}
				<button
					type="button"
					class="btn-ghost text-zinc-500 hover:bg-rose-50 hover:text-rose-600"
					disabled={deleteProduct.isPending}
					onclick={remove}
				>
					<Icon name="trash" size={16} />{m.common_delete()}
				</button>
			{/if}
			{#if siteUrl}
				<!-- eslint-disable svelte/no-navigation-without-resolve -- external storefront URL -->
				<a class="btn-secondary" href={siteUrl} target="_blank" rel="noopener noreferrer">
					<Icon name="external" size={16} />{m.products_view_on_site()}
				</a>
				<!-- eslint-enable svelte/no-navigation-without-resolve -->
			{/if}
		{/snippet}
	</PageHeader>
	{#key product.id}
		<ProductForm
			{product}
			categories={categoriesList.data}
			cover={imageUrl(product.images[0], 800)}
		>
			{#snippet media()}
				<ProductImages product={product!} />
			{/snippet}
		</ProductForm>
	{/key}
{/if}
