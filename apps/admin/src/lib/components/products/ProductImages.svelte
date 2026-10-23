<script lang="ts">
	import type { Image, Product } from '@atcreations/api-types';
	import { useQueryClient } from '@tanstack/svelte-query';
	import { uploadWithProgress } from '$lib/api';
	import Icon from '$lib/components/Icon.svelte';
	import { describeError } from '$lib/i18n';
	import { safeImageUrl } from '$lib/links';
	import { toastError } from '$lib/notify';
	import { m } from '$lib/paraglide/messages.js';
	import {
		createRemoveImageMutation,
		createReorderImagesMutation,
		keys,
		putProduct
	} from '$lib/queries';
	import { confirmAction } from '$lib/state/confirm.svelte';
	import { getAdminSession } from '$lib/state/session.svelte';
	import { toast } from '$lib/state/toast.svelte';
	import {
		checkImageFile,
		IMAGE_TYPES,
		MAX_IMAGE_MB,
		MAX_PRODUCT_IMAGES,
		moveItem
	} from '$lib/utils/files';

	/** The cached product; every change goes back through the query cache. */
	let { product }: { product: Product } = $props();

	const session = getAdminSession();
	const queryClient = useQueryClient();
	const reorder = createReorderImagesMutation();
	const removeImg = createRemoveImageMutation();
	const canEdit = $derived(session.can('products:write'));
	const uid = $props.id();

	type QueueStatus = 'queued' | 'uploading' | 'done' | 'failed';
	interface QueueItem {
		key: number;
		file: File;
		status: QueueStatus;
		progress: number;
		error?: string;
	}

	let queue = $state<QueueItem[]>([]);
	let uploading = $state(false);
	const reordering = $derived(reorder.isPending);
	let dragIndex = $state<number | null>(null);
	let dropIndex = $state<number | null>(null);
	let dropActive = $state(false);
	let fileInput = $state<HTMLInputElement>();
	let announcement = $state('');
	let nextKey = 1;

	const images = $derived(product.images);
	const pendingCount = $derived(
		queue.filter((q) => q.status === 'queued' || q.status === 'uploading').length
	);
	const remaining = $derived(MAX_PRODUCT_IMAGES - images.length - pendingCount);

	function rejectionMessage(reason: 'type' | 'size' | 'count') {
		if (reason === 'type') return m.img_invalid_type();
		if (reason === 'size') return m.img_too_large({ size: MAX_IMAGE_MB });
		return m.img_max_exceeded({ max: MAX_PRODUCT_IMAGES });
	}

	function addFiles(files: FileList | File[] | null | undefined) {
		if (!files || !canEdit) return;
		let slots = remaining;
		const added: QueueItem[] = [];
		for (const file of Array.from(files)) {
			const reason = checkImageFile(file) ?? (slots <= 0 ? 'count' : null);
			if (reason) {
				added.push({
					key: nextKey++,
					file,
					status: 'failed',
					progress: 0,
					error: rejectionMessage(reason)
				});
			} else {
				slots--;
				added.push({ key: nextKey++, file, status: 'queued', progress: 0 });
			}
		}
		queue.push(...added);
		void processQueue();
	}

	async function processQueue() {
		if (uploading) return;
		uploading = true;
		let ok = 0;
		let failed = 0;
		try {
			for (
				let item = queue.find((q) => q.status === 'queued');
				item;
				item = queue.find((q) => q.status === 'queued')
			) {
				item.status = 'uploading';
				item.progress = 0;
				const form = new FormData();
				form.append('file', item.file, item.file.name);
				try {
					const updated = await uploadWithProgress<Product>(
						'POST',
						`/v1/admin/products/${encodeURIComponent(product.id)}/images`,
						form,
						(fraction) => (item!.progress = Math.round(fraction * 100))
					);
					// Upload stays imperative (XHR progress); the response is the canonical product.
					if (updated) await putProduct(queryClient, updated);
					else await queryClient.invalidateQueries({ queryKey: keys.products.detail(product.id) });
					item.status = 'done';
					item.progress = 100;
					ok++;
				} catch (error) {
					item.status = 'failed';
					item.error = describeError(error).message;
					failed++;
				}
			}
		} finally {
			uploading = false;
		}
		if (ok > 0) toast.success(m.img_uploaded_summary({ count: ok }));
		if (failed > 0) toast.error(m.img_failed_summary({ count: failed }));
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

	function statusText(item: QueueItem) {
		switch (item.status) {
			case 'queued':
				return m.img_status_queued();
			case 'uploading':
				return m.img_status_uploading({ percent: item.progress });
			case 'done':
				return m.img_status_done();
			default:
				return m.img_status_failed({ reason: item.error ?? '' });
		}
	}
</script>

<section class="card p-5" aria-labelledby="{uid}-title">
	<h2 id="{uid}-title" class="flex items-baseline justify-between gap-3 card-title">
		{m.img_title()}
		<span class="text-xs font-normal text-zinc-400 tabular-nums"
			>{m.img_count({ current: images.length, max: MAX_PRODUCT_IMAGES })}</span
		>
	</h2>
	<p class="mt-1 text-xs text-zinc-500">{m.img_reorder_hint()}</p>
	<p class="sr-only" aria-live="polite">{announcement}</p>

	{#if images.length === 0}
		<p class="mt-4 text-sm text-zinc-500">{m.img_none()}</p>
	{:else}
		<ol class="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-3 xl:grid-cols-4" aria-busy={reordering}>
			{#each images as image, index (image.id)}
				{@const src = safeImageUrl(image.url)}
				<li
					draggable={canEdit && !reordering}
					class={[
						'group relative overflow-hidden rounded-lg border bg-zinc-50',
						dropIndex === index && dragIndex !== index
							? 'border-brand-400 ring-2 ring-brand-200'
							: 'border-zinc-200',
						dragIndex === index && 'opacity-50'
					]}
					ondragstart={(e) => {
						dragIndex = index;
						e.dataTransfer?.setData('text/plain', image.id);
						if (e.dataTransfer) e.dataTransfer.effectAllowed = 'move';
					}}
					ondragover={(e) => {
						if (dragIndex === null) return;
						e.preventDefault();
						dropIndex = index;
					}}
					ondragleave={() => {
						if (dropIndex === index) dropIndex = null;
					}}
					ondrop={(e) => {
						e.preventDefault();
						if (dragIndex !== null && dragIndex !== index) move(dragIndex, index);
						dragIndex = dropIndex = null;
					}}
					ondragend={() => (dragIndex = dropIndex = null)}
				>
					{#if src}
						<img
							{src}
							alt={m.img_alt({ index: index + 1 })}
							class="aspect-square w-full object-cover"
							loading="lazy"
							draggable="false"
						/>
					{/if}
					{#if index === 0}
						<span class="absolute top-2 left-2 badge bg-zinc-900/80 text-white backdrop-blur"
							>{m.img_cover()}</span
						>
					{/if}
					{#if canEdit}
						<span
							class="absolute top-2 right-2 hidden rounded bg-white/90 p-1 text-zinc-500 group-hover:block"
							aria-hidden="true"><Icon name="grip" size={14} /></span
						>
						<div
							class="flex items-center justify-between gap-1 border-t border-zinc-200 bg-white p-1"
						>
							<div class="flex">
								<button
									id="{uid}-img-{index}"
									type="button"
									class="btn-ghost btn-sm"
									aria-label="{m.common_move_up()} — {m.img_alt({ index: index + 1 })}"
									disabled={index === 0 || reordering}
									onclick={() => move(index, index - 1)}
								>
									<Icon name="left" size={14} />
								</button>
								<button
									type="button"
									class="btn-ghost btn-sm"
									aria-label="{m.common_move_down()} — {m.img_alt({ index: index + 1 })}"
									disabled={index === images.length - 1 || reordering}
									onclick={() => move(index, index + 1)}
								>
									<Icon name="right" size={14} />
								</button>
							</div>
							<button
								type="button"
								class="btn-ghost btn-sm text-red-600 hover:bg-red-50"
								aria-label="{m.img_remove()} — {m.img_alt({ index: index + 1 })}"
								onclick={() => removeImage(image)}
							>
								<Icon name="trash" size={14} />
							</button>
						</div>
					{/if}
				</li>
			{/each}
		</ol>
	{/if}

	{#if canEdit}
		<div
			role="presentation"
			class={[
				'mt-4 flex flex-col items-center justify-center gap-2 rounded-lg border-2 border-dashed p-6 text-center transition',
				dropActive ? 'border-brand-400 bg-brand-50' : 'border-zinc-300',
				remaining <= 0 && 'opacity-60'
			]}
			ondragover={(e) => {
				if (dragIndex !== null || !e.dataTransfer?.types.includes('Files')) return;
				e.preventDefault();
				dropActive = true;
			}}
			ondragleave={() => (dropActive = false)}
			ondrop={(e) => {
				if (dragIndex !== null) return;
				e.preventDefault();
				dropActive = false;
				addFiles(e.dataTransfer?.files);
			}}
		>
			<Icon name="upload" size={28} class="text-zinc-400" />
			{#if remaining <= 0}
				<p class="text-sm font-medium text-zinc-600">{m.img_limit()}</p>
			{:else}
				<p class="text-sm text-zinc-600">{m.img_drop()}</p>
				<button type="button" class="btn-secondary" onclick={() => fileInput?.click()}>
					{m.img_choose()}
				</button>
			{/if}
			<p class="text-xs text-zinc-500">{m.img_specs({ size: MAX_IMAGE_MB })}</p>
			<input
				bind:this={fileInput}
				type="file"
				class="sr-only"
				tabindex="-1"
				aria-hidden="true"
				multiple
				accept={IMAGE_TYPES.join(',')}
				onchange={(e) => {
					addFiles(e.currentTarget.files);
					e.currentTarget.value = '';
				}}
			/>
		</div>
	{/if}

	{#if queue.length > 0}
		<div class="mt-4">
			<div class="mb-2 flex items-center justify-between">
				<h3 class="text-sm font-semibold">{m.img_queue_title()}</h3>
				{#if !uploading}
					<button type="button" class="btn-ghost btn-sm" onclick={() => (queue = [])}
						>{m.img_clear_queue()}</button
					>
				{/if}
			</div>
			<ul class="space-y-2" aria-live="polite">
				{#each queue as item (item.key)}
					<li class="rounded-lg border border-zinc-200 p-2 text-sm">
						<div class="flex items-center justify-between gap-2">
							<span class="min-w-0 truncate font-medium">{item.file.name}</span>
							<span
								class={[
									'shrink-0 text-xs',
									item.status === 'failed' && 'text-red-600',
									item.status === 'done' && 'text-emerald-700',
									(item.status === 'queued' || item.status === 'uploading') && 'text-zinc-500'
								]}
							>
								{statusText(item)}
							</span>
						</div>
						{#if item.status === 'uploading'}
							<progress class="mt-1 h-1.5 w-full" max="100" value={item.progress}
								>{item.progress}%</progress
							>
						{/if}
					</li>
				{/each}
			</ul>
		</div>
	{/if}
</section>
