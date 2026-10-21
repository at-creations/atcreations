<script lang="ts">
	import type { Category } from '@atcreations/api-types';
	import { tick } from 'svelte';
	import { api, unwrap, uploadWithProgress } from '$lib/api';
	import CategoryDialog from '$lib/components/categories/CategoryDialog.svelte';
	import Icon from '$lib/components/Icon.svelte';
	import LoadState from '$lib/components/LoadState.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import { localized } from '$lib/i18n';
	import { safeImageUrl } from '$lib/links';
	import { toastError } from '$lib/notify';
	import { m } from '$lib/paraglide/messages.js';
	import { confirmAction } from '$lib/state/confirm.svelte';
	import { getAdminSession } from '$lib/state/session.svelte';
	import { toast } from '$lib/state/toast.svelte';
	import { checkImageFile, IMAGE_TYPES, MAX_IMAGE_MB } from '$lib/utils/files';

	const session = getAdminSession();
	const canWrite = $derived(session.can('categories:write'));

	let categories = $state<Category[]>([]);
	let loading = $state(true);
	let loadError = $state<unknown>(null);
	let dialogOpen = $state(false);
	let editing = $state<Category | null>(null);
	let dialog = $state<ReturnType<typeof CategoryDialog>>();
	let busyImage = $state<string | null>(null);

	const sorted = $derived(
		[...categories].sort((a, b) => a.sortOrder - b.sortOrder || a.name.en.localeCompare(b.name.en))
	);
	const nextSortOrder = $derived(
		categories.reduce((max, c) => Math.max(max, c.sortOrder), -10) + 10
	);

	async function load() {
		loading = true;
		loadError = null;
		try {
			categories = (await unwrap(api().GET('/v1/admin/categories'))).data;
		} catch (error) {
			loadError = error;
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		load();
	});

	async function openDialog(category: Category | null) {
		editing = category;
		await tick();
		dialog?.reset();
		dialogOpen = true;
	}

	function upsert(category: Category) {
		const index = categories.findIndex((c) => c.id === category.id);
		if (index === -1) categories.push(category);
		else categories[index] = category;
	}

	async function remove(category: Category) {
		const name = localized(category.name);
		const ok = await confirmAction({
			title: m.cat_delete_title(),
			message: m.cat_delete_message({ name }),
			confirmLabel: m.common_delete(),
			danger: true
		});
		if (!ok) return;
		try {
			await unwrap(
				api().DELETE('/v1/admin/categories/{id}', { params: { path: { id: category.id } } })
			);
			categories = categories.filter((c) => c.id !== category.id);
			toast.success(m.cat_deleted({ name }));
		} catch (error) {
			toastError(error, m.cat_delete_failed());
		}
	}

	async function uploadImage(category: Category, file: File | undefined) {
		if (!file) return;
		const rejection = checkImageFile(file);
		if (rejection) {
			toast.error(
				rejection === 'type' ? m.img_invalid_type() : m.img_too_large({ size: MAX_IMAGE_MB })
			);
			return;
		}
		busyImage = category.id;
		try {
			const form = new FormData();
			form.append('file', file, file.name);
			const updated = await uploadWithProgress<Category>(
				'PUT',
				`/v1/admin/categories/${encodeURIComponent(category.id)}/image`,
				form
			);
			if (updated) upsert(updated);
			toast.success(m.cat_image_uploaded());
		} catch (error) {
			toastError(error, m.cat_image_upload_failed());
		} finally {
			busyImage = null;
		}
	}

	async function removeImage(category: Category) {
		const ok = await confirmAction({
			title: m.cat_image_remove_title(),
			message: m.cat_image_remove_message(),
			confirmLabel: m.cat_remove(),
			danger: true
		});
		if (!ok) return;
		busyImage = category.id;
		try {
			const res = await unwrap(
				api().DELETE('/v1/admin/categories/{id}/image', { params: { path: { id: category.id } } })
			);
			upsert(res.data);
			toast.success(m.cat_image_removed());
		} catch (error) {
			toastError(error, m.cat_image_remove_failed());
		} finally {
			busyImage = null;
		}
	}
</script>

<PageHeader title={m.cat_title()}>
	{#snippet actions()}
		{#if canWrite}
			<button type="button" class="btn-primary" onclick={() => openDialog(null)}>
				<Icon name="plus" />{m.cat_add()}
			</button>
		{/if}
	{/snippet}
</PageHeader>

{#if loading && categories.length === 0}
	<div class="card"><LoadState loading /></div>
{:else if loadError}
	<div class="card">
		<LoadState error={loadError} fallback={m.cat_load_failed()} onretry={load} />
	</div>
{:else if sorted.length === 0}
	<p class="card p-10 text-center text-slate-500">{m.cat_none()}</p>
{:else}
	<ul class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
		{#each sorted as category (category.id)}
			{@const src = safeImageUrl(category.image?.url)}
			{@const busy = busyImage === category.id}
			<li class="flex flex-col overflow-hidden card">
				<div class="relative aspect-[16/9] bg-slate-100">
					{#if src}
						<img
							{src}
							alt={m.cat_image_alt({ name: localized(category.name) })}
							class="h-full w-full object-cover"
							loading="lazy"
						/>
					{:else}
						<div class="flex h-full flex-col items-center justify-center gap-1 text-slate-400">
							<Icon name="image" size={28} />
							<span class="text-xs">{m.cat_no_image()}</span>
						</div>
					{/if}
					<span class="absolute top-2 left-2 badge bg-white/90 text-slate-700"
						>{m.cat_order_label({ order: category.sortOrder })}</span
					>
					{#if busy}
						<div
							class="absolute inset-0 flex items-center justify-center bg-white/70 text-sm"
							role="status"
						>
							{m.cat_uploading()}
						</div>
					{/if}
				</div>
				<div class="flex flex-1 flex-col gap-2 p-4">
					<div>
						<h2 class="font-semibold">{category.name.en}</h2>
						{#if category.name.vi}<p class="text-sm text-slate-600" lang="vi">
								{category.name.vi}
							</p>{/if}
						<p class="font-mono text-xs text-slate-400">{category.slug}</p>
					</div>
					<p class="line-clamp-3 text-sm text-slate-700">{category.description.en}</p>
					<p class="line-clamp-3 text-sm text-slate-500" lang="vi">
						{category.description.vi || m.cat_no_vi()}
					</p>
					{#if canWrite}
						<div class="mt-auto flex flex-wrap gap-2 border-t border-slate-100 pt-3">
							<label
								class={[
									'btn-secondary cursor-pointer btn-sm focus-within:ring-2 focus-within:ring-brand-600',
									busy && 'pointer-events-none opacity-50'
								]}
							>
								<Icon name="upload" size={14} />
								{src ? m.cat_change() : m.cat_upload()}
								<input
									type="file"
									class="sr-only"
									accept={IMAGE_TYPES.join(',')}
									disabled={busy}
									onchange={(e) => {
										uploadImage(category, e.currentTarget.files?.[0]);
										e.currentTarget.value = '';
									}}
								/>
							</label>
							{#if src}
								<button
									type="button"
									class="btn-ghost btn-sm"
									disabled={busy}
									onclick={() => removeImage(category)}>{m.cat_remove()}</button
								>
							{/if}
							<span class="flex-1"></span>
							<button
								type="button"
								class="btn-secondary btn-sm"
								onclick={() => openDialog(category)}
							>
								<Icon name="edit" size={14} />{m.common_edit()}
							</button>
							<button
								type="button"
								class="btn-ghost btn-sm text-red-600 hover:bg-red-50"
								onclick={() => remove(category)}
							>
								<Icon name="trash" size={14} />{m.common_delete()}
							</button>
						</div>
					{/if}
				</div>
			</li>
		{/each}
	</ul>
{/if}

<CategoryDialog
	bind:this={dialog}
	bind:open={dialogOpen}
	category={editing}
	{nextSortOrder}
	onsaved={upsert}
/>
