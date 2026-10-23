<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import CategoryForm from '$lib/components/categories/CategoryForm.svelte';
	import Icon from '$lib/components/Icon.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import RequirePermission from '$lib/components/RequirePermission.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import { m } from '$lib/paraglide/messages.js';
	import { categoriesQuery } from '$lib/queries';

	const list = createQuery(() => categoriesQuery());
	// New categories go to the end of the list.
	const nextSortOrder = $derived(
		(list.data ?? []).reduce((max, c) => Math.max(max, c.sortOrder), 0) + 10
	);
</script>

<RequirePermission permission="categories:write">
	<PageHeader
		title={m.cat_create_title()}
		back={{ href: resolve('/categories'), label: m.nav_categories() }}
	/>
	<CategoryForm
		{nextSortOrder}
		onsaved={async (category) => {
			// Continue on the edit page, where the image can be uploaded.
			await goto(resolve(`/categories/${category.id}`), { replaceState: true });
		}}
	>
		{#snippet aside()}
			<Card title={m.cat_image()}>
				<div
					class="flex aspect-[4/3] flex-col items-center justify-center gap-2 rounded-xl border-2 border-dashed border-zinc-200 px-4 text-center"
				>
					<span
						class="flex size-10 items-center justify-center rounded-full bg-zinc-100 text-zinc-400"
					>
						<Icon name="image" size={18} />
					</span>
					<p class="text-sm text-zinc-500">{m.cat_image_after_save()}</p>
				</div>
			</Card>
		{/snippet}
	</CategoryForm>
</RequirePermission>
