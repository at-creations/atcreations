<script lang="ts">
	import type { AdminCategory, UploadJob } from '@atcreations/api-types';
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { onDestroy, untrack } from 'svelte';
	import Icon from '$lib/components/Icon.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Thumb from '$lib/components/ui/Thumb.svelte';
	import { localized } from '$lib/i18n';
	import { toastError } from '$lib/notify';
	import { m } from '$lib/paraglide/messages.js';
	import {
		createCancelUploadMutation,
		createRemoveCategoryImageMutation,
		isActiveJob,
		keys,
		uploadJobsQuery
	} from '$lib/queries';
	import { confirmAction } from '$lib/state/confirm.svelte';
	import { getAdminSession } from '$lib/state/session.svelte';
	import { toast } from '$lib/state/toast.svelte';
	import { uploadItems, type UploadItem } from '$lib/uploads/items';
	import { MAX_UPLOAD_MB, Uploader, UPLOAD_TYPES } from '$lib/uploads/uploader.svelte';
	import UploadTile from '$lib/uploads/UploadTile.svelte';

	let { category }: { category: AdminCategory } = $props();

	const session = getAdminSession();
	const queryClient = useQueryClient();
	const canEdit = $derived(session.can('categories:write'));
	const removeImage = createRemoveCategoryImageMutation();
	const cancelUpload = createCancelUploadMutation('category', () => category.id);
	const jobs = createQuery(() => uploadJobsQuery('category', category.id));

	const uploader = new Uploader(
		'category',
		untrack(() => category.id),
		(job, fileId) => {
			queryClient.setQueryData<UploadJob[]>(keys.uploads.list('category', category.id), (list) =>
				list?.some((j) => j.id === job.id) ? list : [...(list ?? []), job]
			);
			void queryClient.invalidateQueries({ queryKey: keys.uploads.list('category', category.id) });
			uploader.dismiss(fileId);
		}
	);
	onDestroy(() => uploader.destroy());

	// A finished job replaced the image: refresh the category.
	let active = new Set<string>();
	$effect(() => {
		const list = jobs.data ?? [];
		const now = new Set(list.filter(isActiveJob).map((j) => j.id));
		const finished = [...active].filter((id) => !now.has(id));
		active = now;
		if (finished.length === 0) return;
		untrack(() => {
			void queryClient.invalidateQueries({ queryKey: keys.categories.all });
			const failed = list.some((j) => finished.includes(j.id) && j.status === 'failed');
			if (failed) toast.error(m.cat_image_upload_failed());
			else toast.success(m.cat_image_uploaded());
		});
	});

	const items = $derived(uploadItems(uploader.files, jobs.data));
	const busy = $derived(items.some((i) => i.phase !== 'failed'));
	let fileInput = $state<HTMLInputElement>();
	let dropActive = $state(false);

	function addFiles(files: FileList | File[] | null | undefined) {
		if (!files?.length || !canEdit) return;
		// One image per category: only the first file is used.
		uploader.add([files[0]!], 1);
	}

	async function remove() {
		const ok = await confirmAction({
			title: m.cat_image_remove_title(),
			message: m.cat_image_remove_message(),
			confirmLabel: m.cat_remove(),
			danger: true
		});
		if (!ok) return;
		try {
			await removeImage.mutateAsync(category.id);
			toast.success(m.cat_image_removed());
		} catch (error) {
			toastError(error, m.cat_image_remove_failed());
		}
	}

	async function cancel(item: UploadItem) {
		if (item.jobId) {
			try {
				await cancelUpload.mutateAsync(item.jobId);
			} catch (error) {
				toastError(error);
			}
		} else if (item.fileId) {
			uploader.dismiss(item.fileId);
		}
	}
</script>

<Card title={m.cat_image()}>
	{#snippet actions()}
		{#if canEdit && category.image && !busy}
			<button
				type="button"
				class="btn-ghost btn-sm text-zinc-500 hover:text-rose-600"
				disabled={removeImage.isPending}
				onclick={remove}>{m.cat_remove()}</button
			>
		{/if}
	{/snippet}
	<div
		role="presentation"
		class={['relative', dropActive && 'rounded-xl ring-2 ring-brand-300']}
		ondragover={(e) => {
			if (!canEdit || !e.dataTransfer?.types.includes('Files')) return;
			e.preventDefault();
			dropActive = true;
		}}
		ondragleave={() => (dropActive = false)}
		ondrop={(e) => {
			if (!canEdit) return;
			e.preventDefault();
			dropActive = false;
			addFiles(e.dataTransfer?.files);
		}}
	>
		{#if items.length > 0}
			<div class="grid grid-cols-2 gap-2">
				{#if category.image}
					<Thumb
						src={category.image.url}
						class="aspect-square w-full opacity-60"
						rounded="rounded-xl"
						alt={m.cat_image_alt({ name: localized(category.name) })}
					/>
				{/if}
				{#each items as item (item.key)}
					<UploadTile
						{item}
						oncancel={canEdit ? cancel : undefined}
						onretry={(i) => i.fileId && uploader.retry(i.fileId)}
						canRetry={(i) => !!i.fileId && uploader.canRetry(i.fileId)}
					/>
				{/each}
			</div>
		{:else if category.image}
			<button
				type="button"
				class="group relative block w-full overflow-hidden rounded-xl disabled:cursor-default"
				disabled={!canEdit}
				onclick={() => fileInput?.click()}
			>
				<Thumb
					src={category.image.url}
					class="aspect-[4/3] w-full"
					rounded="rounded-xl"
					alt={m.cat_image_alt({ name: localized(category.name) })}
				/>
				{#if canEdit}
					<span
						class="absolute inset-0 flex items-center justify-center gap-2 bg-zinc-900/40 text-sm font-medium text-white opacity-0 transition-opacity group-hover:opacity-100 group-focus-visible:opacity-100"
					>
						<Icon name="upload" size={16} />{m.cat_change()}
					</span>
				{/if}
			</button>
		{:else}
			<button
				type="button"
				class="flex aspect-[4/3] w-full flex-col items-center justify-center gap-2 rounded-xl border-2 border-dashed border-zinc-200 px-4 text-center transition-colors hover:border-brand-300 hover:bg-brand-50/50 disabled:pointer-events-none"
				disabled={!canEdit}
				onclick={() => fileInput?.click()}
			>
				<span
					class="flex size-10 items-center justify-center rounded-full bg-brand-100 text-brand-700"
				>
					<Icon name="upload" size={18} />
				</span>
				<span class="text-sm font-medium text-zinc-700">{m.cat_image_drop()}</span>
				<span class="text-xs text-zinc-400">{m.img_specs({ size: MAX_UPLOAD_MB })}</span>
			</button>
		{/if}
	</div>
	{#if canEdit}
		<input
			bind:this={fileInput}
			type="file"
			class="sr-only"
			tabindex="-1"
			aria-hidden="true"
			accept={UPLOAD_TYPES.join(',')}
			onchange={(e) => {
				addFiles(e.currentTarget.files);
				e.currentTarget.value = '';
			}}
		/>
	{/if}
</Card>
