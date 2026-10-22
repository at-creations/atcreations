<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import Icon from '$lib/components/Icon.svelte';
	import LoadState from '$lib/components/LoadState.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import ProductForm from '$lib/components/products/ProductForm.svelte';
	import RequirePermission from '$lib/components/RequirePermission.svelte';
	import { m } from '$lib/paraglide/messages.js';
	import { categoriesQuery } from '$lib/queries';

	const categories = createQuery(() => categoriesQuery());
</script>

<RequirePermission permission="products:write">
	<a
		href={resolve('/products')}
		class="mb-4 inline-flex items-center gap-1 text-sm text-slate-600 hover:underline"
	>
		<Icon name="left" size={16} />{m.nav_products()}
	</a>
	<PageHeader title={m.pf_title_add()} />
	{#if categories.isPending}
		<LoadState loading />
	{:else if categories.isError && !categories.data}
		<LoadState
			error={categories.error}
			fallback={m.cat_load_failed()}
			onretry={() => categories.refetch()}
		/>
	{:else}
		<ProductForm
			categories={categories.data ?? []}
			onsaved={async (product) => {
				// Step 2: continue to the edit page where images can be uploaded (already cached).
				await goto(resolve(`/products/${product.id}`), { replaceState: true });
			}}
		/>
		<p class="mt-6 flex items-center gap-2 rounded-lg bg-sky-50 p-4 text-sm text-sky-900">
			<Icon name="info" />
			{m.pf_images_after_save()}
		</p>
	{/if}
</RequirePermission>
