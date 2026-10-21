<script lang="ts">
	import type { Category, Product } from '@atcreations/api-types';
	import { untrack } from 'svelte';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { api, ApiError, unwrap } from '$lib/api';
	import Icon from '$lib/components/Icon.svelte';
	import LoadState from '$lib/components/LoadState.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import ProductForm from '$lib/components/products/ProductForm.svelte';
	import ProductImages from '$lib/components/products/ProductImages.svelte';
	import StatusBadge from '$lib/components/StatusBadge.svelte';
	import { localized } from '$lib/i18n';
	import { storefrontProductUrl } from '$lib/links';
	import { m } from '$lib/paraglide/messages.js';
	import { getAdminSession } from '$lib/state/session.svelte';

	const session = getAdminSession();
	let product = $state<Product | null>(null);
	let categories = $state<Category[]>([]);
	let loading = $state(true);
	let loadError = $state<unknown>(null);
	let notFound = $state(false);

	async function load(id: string) {
		loading = true;
		loadError = null;
		notFound = false;
		product = null;
		try {
			const [p, c] = await Promise.all([
				unwrap(api().GET('/v1/admin/products/{id}', { params: { path: { id } } })),
				unwrap(api().GET('/v1/admin/categories'))
			]);
			product = p.data;
			categories = c.data;
		} catch (error) {
			if (error instanceof ApiError && (error.status === 404 || error.status === 400))
				notFound = true;
			else loadError = error;
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		const id = page.params.id ?? '';
		untrack(() => load(id));
	});

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

{#if loading}
	<LoadState loading />
{:else if notFound}
	<div class="card p-10 text-center">
		<p class="font-medium">{m.pf_not_found()}</p>
		<a class="mt-4 btn-secondary" href={resolve('/products')}>{m.common_back()}</a>
	</div>
{:else if loadError || !product}
	<LoadState error={loadError} onretry={() => load(page.params.id ?? '')} />
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
			<ProductForm
				{product}
				{categories}
				onsaved={(saved) => {
					product = { ...saved };
				}}
			/>
		{/key}
		<ProductImages bind:product={() => product!, (value) => (product = value)} />
	</div>
{/if}
