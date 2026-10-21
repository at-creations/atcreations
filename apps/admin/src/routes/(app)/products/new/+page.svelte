<script lang="ts">
	import type { Category } from '@atcreations/api-types';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { api, unwrap } from '$lib/api';
	import Icon from '$lib/components/Icon.svelte';
	import LoadState from '$lib/components/LoadState.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import ProductForm from '$lib/components/products/ProductForm.svelte';
	import RequirePermission from '$lib/components/RequirePermission.svelte';
	import { m } from '$lib/paraglide/messages.js';

	let categories = $state<Category[] | null>(null);
	let loadError = $state<unknown>(null);

	async function loadCategories() {
		loadError = null;
		try {
			categories = (await unwrap(api().GET('/v1/admin/categories'))).data;
		} catch (error) {
			loadError = error;
		}
	}

	$effect(() => {
		loadCategories();
	});
</script>

<RequirePermission permission="products:write">
	<a
		href={resolve('/products')}
		class="mb-4 inline-flex items-center gap-1 text-sm text-slate-600 hover:underline"
	>
		<Icon name="left" size={16} />{m.nav_products()}
	</a>
	<PageHeader title={m.pf_title_add()} />
	{#if loadError}
		<LoadState error={loadError} fallback={m.cat_load_failed()} onretry={loadCategories} />
	{:else if !categories}
		<LoadState loading />
	{:else}
		<ProductForm
			{categories}
			onsaved={async (product) => {
				// Step 2: continue to the edit page where images can be uploaded.
				await goto(resolve(`/products/${product.id}`), { replaceState: true });
			}}
		/>
		<p class="mt-6 flex items-center gap-2 rounded-lg bg-sky-50 p-4 text-sm text-sky-900">
			<Icon name="info" />
			{m.pf_images_after_save()}
		</p>
	{/if}
</RequirePermission>
