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
	<PageHeader
		title={m.pf_title_add()}
		back={{ href: resolve('/products'), label: m.nav_products() }}
	/>
	{#if categories.isPending}
		<LoadState loading />
	{:else if categories.isError && !categories.data}
		<div class="card">
			<LoadState
				error={categories.error}
				fallback={m.cat_load_failed()}
				onretry={() => categories.refetch()}
			/>
		</div>
	{:else}
		<ProductForm
			categories={categories.data ?? []}
			onsaved={async (product) => {
				// Step 2: continue to the edit page where images can be uploaded (already cached).
				await goto(resolve(`/products/${product.id}`), { replaceState: true });
			}}
		>
			{#snippet media()}
				<section class="card">
					<div class="card-header">
						<h2 class="card-title">{m.img_title()}</h2>
					</div>
					<div
						class="m-5 flex flex-col items-center gap-2 rounded-lg border-2 border-dashed border-zinc-200 px-6 py-10 text-center"
					>
						<Icon name="image" size={24} class="text-zinc-300" />
						<p class="text-sm text-zinc-500">{m.pf_images_after_save()}</p>
					</div>
				</section>
			{/snippet}
		</ProductForm>
	{/if}
</RequirePermission>
