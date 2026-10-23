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
		{#if images.length === 0}
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
				<span class="text-xs text-zinc-400">{m.img_specs({ size: MAX_IMAGE_MB })}</span>
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
										aria-label="{m.common_move_up()} — {m.img_alt({ index: index + 1 })}"
										disabled={index === 0 || reordering}
										onclick={() => move(index, index - 1)}
									>
										<Icon name="left" size={14} />
									</button>
									<button
										type="button"
										class="btn-icon btn-ghost size-7"
										aria-label="{m.common_move_down()} — {m.img_alt({ index: index + 1 })}"
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

		{#if queue.length > 0}
			<div class="mt-4 rounded-xl bg-zinc-50 p-3">
				<div class="mb-2 flex items-center justify-between">
					<h3 class="text-xs font-semibold text-zinc-700">{m.img_queue_title()}</h3>
					{#if !uploading}
						<button
							type="button"
							class="text-xs text-zinc-500 hover:text-zinc-900"
							onclick={() => (queue = [])}>{m.img_clear_queue()}</button
						>
					{/if}
				</div>
				<ul class="space-y-2" aria-live="polite">
					{#each queue as item (item.key)}
						<li class="text-xs">
							<div class="flex items-center justify-between gap-2">
								<span class="min-w-0 truncate text-zinc-700">{item.file.name}</span>
								<span
									class={[
										'shrink-0',
										item.status === 'failed' && 'text-rose-600',
										item.status === 'done' && 'text-emerald-700',
										(item.status === 'queued' || item.status === 'uploading') && 'text-zinc-500'
									]}
								>
									{statusText(item)}
								</span>
							</div>
							{#if item.status === 'uploading'}
								<div class="mt-1 h-1 overflow-hidden rounded-full bg-zinc-200">
									<div
										class="h-full rounded-full bg-brand-400 transition-[width]"
										style:width="{item.progress}%"
									></div>
								</div>
							{/if}
						</li>
					{/each}
				</ul>
			</div>
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
			accept={IMAGE_TYPES.join(',')}
			onchange={(e) => {
				addFiles(e.currentTarget.files);
				e.currentTarget.value = '';
			}}
		/>
	{/if}
</section>
