<script lang="ts">
	import type { Image, Product, UploadJob } from '@atcreations/api-types';
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { onDestroy, untrack } from 'svelte';
	import Icon from '$lib/components/Icon.svelte';
	import { safeImageUrl } from '$lib/links';
	import { toastError } from '$lib/notify';
	import { m } from '$lib/paraglide/messages.js';
	import {
		createCancelUploadMutation,
		createRemoveImageMutation,
		createReorderImagesMutation,
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
	import { MAX_PRODUCT_IMAGES, moveItem } from '$lib/utils/files';

	/** The cached product; every change goes back through the query cache. */
	let { product }: { product: Product } = $props();

	const session = getAdminSession();
	const queryClient = useQueryClient();
	const reorder = createReorderImagesMutation();
	const removeImg = createRemoveImageMutation();
	const cancelUpload = createCancelUploadMutation('product', () => product.id);
	const canEdit = $derived(session.can('products:write'));
	const uid = $props.id();

	const jobs = createQuery(() => uploadJobsQuery('product', product.id));

	// One uploader per product page (this component is keyed by product).
	const uploader = new Uploader(
		'product',
		untrack(() => product.id),
		(job, fileId) => {
			// Show the server job right away (before the next poll) and retire the local tile.
			queryClient.setQueryData<UploadJob[]>(keys.uploads.list('product', product.id), (list) =>
				list?.some((j) => j.id === job.id) ? list : [...(list ?? []), job]
			);
			void queryClient.invalidateQueries({ queryKey: keys.uploads.list('product', product.id) });
			uploader.dismiss(fileId);
		}
	);
	onDestroy(() => uploader.destroy());

	// When a job finishes, the product gains an image: refresh it.
	let active = new Set<string>();
	$effect(() => {
		const list = jobs.data ?? [];
		const now = new Set(list.filter(isActiveJob).map((j) => j.id));
		const finished = [...active].filter((id) => !now.has(id));
		active = now;
		if (finished.length === 0) return;
		untrack(() => {
			void queryClient.invalidateQueries({ queryKey: keys.products.detail(product.id) });
			void queryClient.invalidateQueries({ queryKey: keys.products.lists() });
			const failed = list.filter((j) => finished.includes(j.id) && j.status === 'failed').length;
			const done = finished.length - failed;
			if (done > 0) toast.success(m.up_done_summary({ count: done }));
			if (failed > 0) toast.error(m.img_failed_summary({ count: failed }));
		});
	});

	const reordering = $derived(reorder.isPending);
	let dragIndex = $state<number | null>(null);
	let dropIndex = $state<number | null>(null);
	let dropActive = $state(false);
	let fileInput = $state<HTMLInputElement>();
	let announcement = $state('');

	const images = $derived(product.images);
	const items = $derived(uploadItems(uploader.files, jobs.data));
	const inFlight = $derived(items.filter((i) => i.phase !== 'failed').length);
	const remaining = $derived(MAX_PRODUCT_IMAGES - images.length - inFlight);

	function addFiles(files: FileList | File[] | null | undefined) {
		if (!files || !canEdit) return;
		const list = Array.from(files);
		const added = uploader.add(list, Math.max(remaining, 0));
		if (added < list.length) toast.error(m.img_max_exceeded({ max: MAX_PRODUCT_IMAGES }));
	}

	async function saveOrder(next: Image[], focusIndex?: number) {
		try {
			// Optimistic in the cache; rolled back by the mutation on failure.
			await reorder.mutateAsync({ id: product.id, images: next });
			announcement = m.img_order_saved();
		} catch (error) {
			toastError(error, m.img_order_failed());
		}
		if (focusIndex !== undefined) {
			document.getElementById(`${uid}-img-${focusIndex}`)?.focus();
		}
	}

	function move(from: number, to: number) {
		if (to < 0 || to >= images.length || reordering) return;
		void saveOrder(moveItem(images, from, to), to);
	}

	async function removeImage(image: Image) {
		const ok = await confirmAction({
			title: m.img_remove_title(),
			message: m.img_remove_message(),
			confirmLabel: m.common_delete(),
			danger: true
		});
		if (!ok) return;
		try {
			await removeImg.mutateAsync({ id: product.id, imageId: image.id });
			toast.success(m.img_removed());
		} catch (error) {
			toastError(error, m.img_remove_failed());
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

<section
	class={[
		'relative card transition-shadow',
		dropActive && 'ring-2 ring-brand-300 ring-offset-2 ring-offset-canvas'
	]}
	aria-labelledby="{uid}-title"
	ondragover={(e) => {
		if (!canEdit || dragIndex !== null || !e.dataTransfer?.types.includes('Files')) return;
		e.preventDefault();
		dropActive = true;
	}}
	ondragleave={(e) => {
		if (!e.currentTarget.contains(e.relatedTarget as Node | null)) dropActive = false;
	}}
	ondrop={(e) => {
		if (!canEdit || dragIndex !== null) return;
		e.preventDefault();
		dropActive = false;
		addFiles(e.dataTransfer?.files);
	}}
>
	<header class="card-header">
		<div class="min-w-0">
			<h2 id="{uid}-title" class="card-title">
				{m.img_title()}
				<span class="ml-1 text-xs font-normal text-zinc-400 tabular-nums"
					>{images.length}/{MAX_PRODUCT_IMAGES}</span
				>
			</h2>
			{#if images.length > 1 && canEdit}
				<p class="mt-0.5 text-xs text-zinc-500">{m.img_reorder_hint()}</p>
			{/if}
		</div>
		{#if canEdit && remaining > 0}
			<button type="button" class="btn-secondary btn-sm" onclick={() => fileInput?.click()}>
				<Icon name="upload" size={14} />{m.img_choose()}
			</button>
		{/if}
	</header>
	<p class="sr-only" aria-live="polite">{announcement}</p>

	<div class="p-5">
		{#if images.length === 0 && items.length === 0}
			<button
				type="button"
				class="flex w-full flex-col items-center justify-center gap-2 rounded-xl border-2 border-dashed border-zinc-200 px-6 py-12 text-center transition-colors hover:border-brand-300 hover:bg-brand-50/50 disabled:pointer-events-none"
				disabled={!canEdit}
				onclick={() => fileInput?.click()}
			>
				<span
					class="flex size-11 items-center justify-center rounded-full bg-brand-100 text-brand-700"
				>
					<Icon name="upload" size={20} />
				</span>
				<span class="text-sm font-medium text-zinc-700">{m.img_drop()}</span>
				<span class="text-xs text-zinc-400">{m.img_specs({ size: MAX_UPLOAD_MB })}</span>
			</button>
		{:else}
			<ol class="grid grid-cols-3 gap-3 sm:grid-cols-4 xl:grid-cols-5" aria-busy={reordering}>
				{#each images as image, index (image.id)}
					{@const src = safeImageUrl(image.url)}
					<li
						draggable={canEdit && !reordering}
						class={[
							'group relative overflow-hidden rounded-xl bg-zinc-100 ring-1 ring-zinc-200 transition',
							index === 0 && 'col-span-2 row-span-2',
							dropIndex === index && dragIndex !== index && 'ring-2 ring-brand-400',
							dragIndex === index && 'opacity-40',
							canEdit && 'cursor-grab active:cursor-grabbing'
						]}
						ondragstart={(e) => {
							dragIndex = index;
							e.dataTransfer?.setData('text/plain', image.id);
							if (e.dataTransfer) e.dataTransfer.effectAllowed = 'move';
						}}
						ondragover={(e) => {
							if (dragIndex === null) return;
							e.preventDefault();
							e.stopPropagation();
							dropIndex = index;
						}}
						ondragleave={() => {
							if (dropIndex === index) dropIndex = null;
						}}
						ondrop={(e) => {
							if (dragIndex === null) return;
							e.preventDefault();
							e.stopPropagation();
							if (dragIndex !== index) move(dragIndex, index);
							dragIndex = dropIndex = null;
						}}
						ondragend={() => (dragIndex = dropIndex = null)}
					>
						{#if src}
							<img
								{src}
								alt={m.img_alt({ index: index + 1 })}
								class="aspect-square size-full object-cover"
								loading="lazy"
								draggable="false"
							/>
						{/if}
						{#if index === 0}
							<span
								class="absolute top-2 left-2 rounded-md bg-white/90 px-2 py-0.5 text-[11px] font-semibold text-zinc-800 shadow-sm"
								>{m.img_cover()}</span
							>
						{/if}
						{#if canEdit}
							<!-- Controls appear on hover/focus; arrows keep reordering keyboard-accessible. -->
							<div
								class="absolute inset-x-1.5 bottom-1.5 flex items-center justify-between gap-1 rounded-lg bg-white/95 p-0.5 opacity-0 shadow-sm transition-opacity group-focus-within:opacity-100 group-hover:opacity-100"
							>
								<div class="flex">
									<button
										id="{uid}-img-{index}"
										type="button"
										class="btn-icon btn-ghost size-7"
										aria-label="{m.img_move_earlier()} — {m.img_alt({ index: index + 1 })}"
										disabled={index === 0 || reordering}
										onclick={() => move(index, index - 1)}
									>
										<Icon name="left" size={14} />
									</button>
									<button
										type="button"
										class="btn-icon btn-ghost size-7"
										aria-label="{m.img_move_later()} — {m.img_alt({ index: index + 1 })}"
										disabled={index === images.length - 1 || reordering}
										onclick={() => move(index, index + 1)}
									>
										<Icon name="right" size={14} />
									</button>
								</div>
								<button
									type="button"
									class="btn-icon btn-ghost size-7 hover:bg-rose-50 hover:text-rose-600"
									aria-label="{m.img_remove()} — {m.img_alt({ index: index + 1 })}"
									onclick={() => removeImage(image)}
								>
									<Icon name="trash" size={14} />
								</button>
							</div>
						{/if}
					</li>
				{/each}
				{#each items as item (item.key)}
					<li>
						<UploadTile
							{item}
							oncancel={canEdit ? cancel : undefined}
							onretry={(i) => i.fileId && uploader.retry(i.fileId)}
							canRetry={(i) => !!i.fileId && uploader.canRetry(i.fileId)}
						/>
					</li>
				{/each}
				{#if canEdit && remaining > 0}
					<li>
						<button
							type="button"
							class="flex aspect-square w-full flex-col items-center justify-center gap-1 rounded-xl border-2 border-dashed border-zinc-200 text-zinc-400 transition-colors hover:border-brand-300 hover:bg-brand-50/50 hover:text-brand-700"
							aria-label={m.img_choose()}
							onclick={() => fileInput?.click()}
						>
							<Icon name="plus" size={20} />
						</button>
					</li>
				{/if}
			</ol>
			{#if canEdit && remaining <= 0}
				<p class="mt-3 text-xs text-zinc-500">{m.img_limit()}</p>
			{/if}
		{/if}
	</div>

	{#if dropActive}
		<div
			class="pointer-events-none absolute inset-0 flex items-center justify-center rounded-2xl bg-brand-50/90 text-sm font-medium text-brand-800"
		>
			<Icon name="upload" size={18} class="mr-2" />{m.img_drop()}
		</div>
	{/if}

	{#if canEdit}
		<input
			bind:this={fileInput}
			type="file"
			class="sr-only"
			tabindex="-1"
			aria-hidden="true"
			multiple
			accept={UPLOAD_TYPES.join(',')}
			onchange={(e) => {
				addFiles(e.currentTarget.files);
				e.currentTarget.value = '';
			}}
		/>
	{/if}
</section>
