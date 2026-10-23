<script lang="ts">
	import type { Category, Product, ProductInput, ProductStatus } from '@atcreations/api-types';
	import { useQueryClient } from '@tanstack/svelte-query';
	import { untrack, type Snippet } from 'svelte';
	import { beforeNavigate, goto } from '$app/navigation';
	import { ApiError, fieldErrorMap } from '$lib/api';
	import Field from '$lib/components/Field.svelte';
	import Icon from '$lib/components/Icon.svelte';
	import LangTabs from '$lib/components/LangTabs.svelte';
	import { localized, priceErrorMessage, productStatusLabel } from '$lib/i18n';
	import { toastError } from '$lib/notify';
	import { m } from '$lib/paraglide/messages.js';
	import { createSaveProductMutation, reloadProduct } from '$lib/queries';
	import { confirmAction } from '$lib/state/confirm.svelte';
	import { getAdminSession } from '$lib/state/session.svelte';
	import { toast } from '$lib/state/toast.svelte';
	import { centsToDollarInput, formatCAD, parseDollars } from '$lib/utils/money';
	import { isValidSlug, slugify } from '$lib/utils/slug';

	let {
		product,
		categories,
		onsaved,
		media
	}: {
		product?: Product;
		categories: Category[];
		/** Called after a successful save; the product cache is already updated. */
		onsaved?: (product: Product, created: boolean) => void | Promise<void>;
		/** Rendered in the main column under the details (the image manager). */
		media?: Snippet;
	} = $props();

	const session = getAdminSession();
	const queryClient = useQueryClient();
	const save = createSaveProductMutation();
	const readOnly = $derived(!session.can('products:write'));
	const uid = $props.id();
	const STATUSES: readonly ProductStatus[] = ['published', 'draft', 'archived'];

	interface FormValues {
		nameEn: string;
		nameVi: string;
		descriptionEn: string;
		descriptionVi: string;
		slug: string;
		status: ProductStatus;
		price: string;
		discount: number | null;
		stock: number | null;
		featured: boolean;
		categoryIds: string[];
	}

	function toValues(p: Product | undefined): FormValues {
		return {
			nameEn: p?.name.en ?? '',
			nameVi: p?.name.vi ?? '',
			descriptionEn: p?.description.en ?? '',
			descriptionVi: p?.description.vi ?? '',
			slug: p?.slug ?? '',
			status: p?.status ?? 'draft',
			price: centsToDollarInput(p?.priceCents),
			discount: p?.discountPercent ?? 0,
			stock: p?.stock ?? 0,
			featured: p?.featured ?? false,
			categoryIds: [...(p?.categoryIds ?? [])]
		};
	}

	// Initial values are captured once; the parent re-mounts this component per product.
	const initial = untrack(() => toValues(product));
	let values = $state<FormValues>(structuredClone(initial));
	let snapshot = $state(JSON.stringify(initial));
	let slugTouched = $state(untrack(() => !!product));
	let errors = $state<Record<string, string>>({});
	let tab = $state<'en' | 'vi'>('en');
	let bypassGuard = false;
	/** The `version` the current form values are based on (sent for optimistic concurrency). */
	let baseVersion = $state(untrack(() => product?.version));
	const saving = $derived(save.isPending);

	const dirty = $derived(JSON.stringify(values) !== snapshot);

	function resetTo(p: Product) {
		const next = toValues(p);
		values = next;
		snapshot = JSON.stringify(next);
		baseVersion = p.version;
		slugTouched = true;
	}

	// The cached product can change underneath the form: our own image operations, a refetch
	// (window focus, invalidation) or a reload after a 409.
	$effect(() => {
		const next = product;
		const isDirty = dirty;
		if (!next || next.version === baseVersion) return;
		untrack(() => {
			if (JSON.stringify(toValues(next)) === snapshot) {
				// Only non-form fields changed (e.g. images): adopt the new version.
				baseVersion = next.version;
			} else if (!isDirty) {
				// Changed elsewhere and nothing to lose: show the latest values.
				resetTo(next);
			}
			// Otherwise keep the old version so saving surfaces a 409 instead of overwriting.
		});
	});
	const parsedPrice = $derived(parseDollars(values.price));
	const finalPreview = $derived.by(() => {
		if (!parsedPrice.ok || parsedPrice.cents === null) return null;
		const d = intInRange(values.discount, 0, 90) ?? 0;
		// Mirrors the API: rounded half-up.
		return Math.floor((parsedPrice.cents * (100 - d)) / 100 + 0.5);
	});

	/** Throw away local edits and return to the last saved values. */
	function discard() {
		values = JSON.parse(snapshot) as FormValues;
		errors = {};
		if (!product) slugTouched = false;
	}

	function onNameEnInput(value: string) {
		values.nameEn = value;
		if (!slugTouched) values.slug = slugify(value);
	}

	function intInRange(n: number | null, min: number, max: number): number | null {
		return n !== null && Number.isInteger(n) && n >= min && n <= max ? n : null;
	}

	function validate(): ProductInput | null {
		const e: Record<string, string> = {};
		if (!values.nameEn.trim()) e['name.en'] = m.validation_required();
		if (!values.descriptionEn.trim()) e['description.en'] = m.validation_required();
		if (values.slug && !isValidSlug(values.slug)) e.slug = m.validation_invalid_slug();
		const priceMsg = priceErrorMessage(parsedPrice);
		if (priceMsg) e.priceCents = priceMsg;
		const discount = intInRange(values.discount, 0, 90);
		if (discount === null) e.discountPercent = m.validation_int_range({ min: 0, max: 90 });
		const stock = intInRange(values.stock, 0, 1_000_000);
		if (stock === null) e.stock = m.validation_int_range({ min: 0, max: 1000000 });
		errors = e;
		if (Object.keys(e).length > 0 || !parsedPrice.ok) {
			if (e['name.en'] || e['description.en']) tab = 'en';
			return null;
		}
		const nameVi = values.nameVi.trim();
		const descriptionVi = values.descriptionVi.trim();
		return {
			slug: values.slug || undefined,
			status: values.status,
			name: { en: values.nameEn.trim(), ...(nameVi ? { vi: nameVi } : {}) },
			description: {
				en: values.descriptionEn.trim(),
				...(descriptionVi ? { vi: descriptionVi } : {})
			},
			priceCents: parsedPrice.cents,
			discountPercent: discount ?? 0,
			stock: stock ?? 0,
			featured: values.featured,
			categoryIds: values.categoryIds,
			// Optimistic concurrency: the API answers 409 if the product (or its
			// stock, via an order) changed since it was loaded.
			...(product ? { version: baseVersion } : {})
		};
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (readOnly || saving) return;
		if (product && !dirty) {
			toast.info(m.pf_no_changes());
			return;
		}
		const body = validate();
		if (!body) {
			toast.error(m.validation_fix_errors());
			return;
		}
		let saved: Product;
		try {
			saved = await save.mutateAsync({ id: product?.id, body });
		} catch (error) {
			errors = fieldErrorMap(error);
			if (errors['name.en'] || errors['description.en']) tab = 'en';
			else if (errors['name.vi'] || errors['description.vi']) tab = 'vi';
			toastError(error, m.pf_save_failed());
			if (product && error instanceof ApiError && error.status === 409)
				void offerReload(product.id);
			return;
		}
		resetTo(saved);
		toast.success(product ? m.pf_updated() : m.pf_created());
		bypassGuard = true;
		try {
			await onsaved?.(saved, !product);
		} finally {
			bypassGuard = false;
		}
	}

	/** 409: the product changed since it was loaded. Offer to discard local edits and reload. */
	async function offerReload(id: string) {
		const ok = await confirmAction({
			title: m.pf_conflict_title(),
			message: m.pf_conflict_message(),
			confirmLabel: m.pf_conflict_reload(),
			danger: true
		});
		if (!ok) return;
		try {
			resetTo(await reloadProduct(queryClient, id));
			errors = {};
			toast.info(m.pf_conflict_reloaded());
		} catch (error) {
			toastError(error);
		}
	}

	beforeNavigate((navigation) => {
		if (!dirty || bypassGuard || readOnly) return;
		if (navigation.type === 'leave') {
			// Browser-native "leave site?" prompt for reloads/closing the tab.
			navigation.cancel();
			return;
		}
		const target = navigation.to?.url;
		navigation.cancel();
		if (!target) return;
		confirmAction({
			title: m.pf_unsaved_title(),
			message: m.pf_unsaved_message(),
			confirmLabel: m.pf_leave(),
			danger: true
		}).then((ok) => {
			if (!ok) return;
			bypassGuard = true;
			// Same-origin URL from SvelteKit's own navigation object.
			// eslint-disable-next-line svelte/no-navigation-without-resolve
			goto(`${target.pathname}${target.search}${target.hash}`).finally(() => (bypassGuard = false));
		});
	});

	function toggleCategory(id: string) {
		values.categoryIds = values.categoryIds.includes(id)
			? values.categoryIds.filter((c) => c !== id)
			: [...values.categoryIds, id];
	}

	const tabErrors = $derived({
		en: !!(errors['name.en'] || errors['description.en']),
		vi: !!(errors['name.vi'] || errors['description.vi'])
	});
</script>

<form onsubmit={submit} novalidate class="pb-24">
	{#if readOnly}
		<p
			class="mb-6 flex items-center gap-2 rounded-lg bg-amber-50 p-3 text-sm text-amber-900 ring-1 ring-amber-200"
		>
			<Icon name="info" size={16} />{m.pf_read_only()}
		</p>
	{/if}

	<div class="grid items-start gap-6 lg:grid-cols-3">
		<!-- Main column -->
		<div class="min-w-0 space-y-6 lg:col-span-2">
			<fieldset disabled={readOnly || saving} class="min-w-0 card">
				<div class="card-header">
					<h2 class="card-title">{m.pf_basic()}</h2>
				</div>
				<div class="p-5">
					<LangTabs bind:active={tab} errors={tabErrors}>
						{#snippet children(lang)}
							{#if lang === 'en'}
								<Field label={m.pf_name()} required error={errors['name.en']}>
									{#snippet children(control)}
										<input
											{...control}
											class="input"
											maxlength="200"
											placeholder={m.pf_name_placeholder()}
											value={values.nameEn}
											oninput={(e) => onNameEnInput(e.currentTarget.value)}
										/>
									{/snippet}
								</Field>
								<Field label={m.pf_description()} required error={errors['description.en']}>
									{#snippet children(control)}
										<textarea
											{...control}
											class="input h-auto min-h-40 py-2"
											maxlength="5000"
											placeholder={m.pf_description_placeholder()}
											bind:value={values.descriptionEn}></textarea>
									{/snippet}
								</Field>
							{:else}
								<p class="flex items-center gap-2 text-xs text-zinc-500">
									<Icon name="info" size={14} />{m.vietnamese_note()}
								</p>
								<Field label={m.pf_name()} optional error={errors['name.vi']}>
									{#snippet children(control)}
										<input
											{...control}
											class="input"
											lang="vi"
											maxlength="200"
											placeholder={m.pf_name_vi_placeholder()}
											bind:value={values.nameVi}
										/>
									{/snippet}
								</Field>
								<Field label={m.pf_description()} optional error={errors['description.vi']}>
									{#snippet children(control)}
										<textarea
											{...control}
											class="input h-auto min-h-40 py-2"
											lang="vi"
											maxlength="5000"
											placeholder={m.pf_description_vi_placeholder()}
											bind:value={values.descriptionVi}></textarea>
									{/snippet}
								</Field>
							{/if}
						{/snippet}
					</LangTabs>
					<div class="mt-4">
						<Field label={m.pf_slug()} error={errors.slug}>
							{#snippet children(control)}
								<div class="flex gap-2">
									<input
										{...control}
										class="input font-mono text-xs"
										maxlength="120"
										autocomplete="off"
										spellcheck="false"
										value={values.slug}
										oninput={(e) => {
											slugTouched = true;
											values.slug = e.currentTarget.value.toLowerCase();
										}}
									/>
									<button
										type="button"
										class="btn-secondary shrink-0"
										onclick={() => (values.slug = slugify(values.nameEn))}
									>
										{m.pf_slug_suggest()}
									</button>
								</div>
							{/snippet}
						</Field>
					</div>
				</div>
			</fieldset>

			{@render media?.()}

			<fieldset disabled={readOnly || saving} class="min-w-0 card">
				<div class="card-header">
					<h2 class="card-title">{m.pf_pricing()}</h2>
				</div>
				<div class="grid gap-4 p-5 sm:grid-cols-2">
					<Field label={m.pf_price()} hint={m.pf_price_hint()} error={errors.priceCents}>
						{#snippet children(control)}
							<div class="relative">
								<span
									class="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-sm text-zinc-400"
									>$</span
								>
								<input
									{...control}
									class="input pl-7 tabular-nums"
									inputmode="decimal"
									placeholder="0.00"
									bind:value={values.price}
								/>
							</div>
						{/snippet}
					</Field>
					<Field label={m.pf_discount()} hint={m.pf_discount_hint()} error={errors.discountPercent}>
						{#snippet children(control)}
							<div class="relative">
								<input
									{...control}
									class="input pr-8 tabular-nums"
									type="number"
									min="0"
									max="90"
									step="1"
									inputmode="numeric"
									bind:value={values.discount}
								/>
								<span
									class="pointer-events-none absolute top-1/2 right-3 -translate-y-1/2 text-sm text-zinc-400"
									>%</span
								>
							</div>
						{/snippet}
					</Field>
				</div>
				<div
					class="flex items-center justify-between rounded-b-xl border-t border-zinc-100 bg-zinc-50/70 px-5 py-3 text-sm"
					aria-live="polite"
				>
					<span class="text-zinc-500">{m.pf_customer_pays()}</span>
					<span class="font-semibold text-zinc-900 tabular-nums">
						{finalPreview === null ? m.products_contact_price() : formatCAD(finalPreview)}
					</span>
				</div>
			</fieldset>
		</div>

		<!-- Side column -->
		<fieldset disabled={readOnly || saving} class="min-w-0 space-y-6">
			<section class="card">
				<div class="card-header">
					<h2 class="card-title" id="{uid}-status-title">{m.pf_status()}</h2>
				</div>
				<div class="space-y-2 p-4" role="radiogroup" aria-labelledby="{uid}-status-title">
					{#each STATUSES as status (status)}
						<label
							class={[
								'flex cursor-pointer items-center gap-3 rounded-lg border px-3 py-2.5 transition-colors has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-brand-600/30',
								values.status === status
									? 'border-brand-300 bg-brand-50'
									: 'border-zinc-200 hover:border-zinc-300'
							]}
						>
							<input
								type="radio"
								name="status"
								value={status}
								class="size-4 accent-brand-500"
								bind:group={values.status}
							/>
							<span class="text-sm font-medium text-zinc-900">{productStatusLabel(status)}</span>
						</label>
					{/each}
					{#if errors.status}<p class="text-xs text-red-600">{errors.status}</p>{/if}
				</div>
			</section>

			<section class="card">
				<div class="card-header">
					<h2 class="card-title">{m.pf_inventory()}</h2>
				</div>
				<div class="p-5">
					<Field label={m.pf_stock()} error={errors.stock}>
						{#snippet children(control)}
							<input
								{...control}
								class="input tabular-nums"
								type="number"
								min="0"
								max="1000000"
								step="1"
								inputmode="numeric"
								bind:value={values.stock}
							/>
						{/snippet}
					</Field>
				</div>
			</section>

			<section class="card">
				<div class="card-header">
					<h2 class="card-title">{m.pf_organization()}</h2>
				</div>
				<div class="space-y-5 p-5">
					<fieldset>
						<legend class="label">
							{m.pf_categories()}
							<span class="font-normal text-zinc-400"
								>{m.pf_categories_selected({ count: values.categoryIds.length })}</span
							>
						</legend>
						{#if categories.length === 0}
							<p class="text-sm text-zinc-500">{m.pf_no_categories()}</p>
						{:else}
							<div class="flex flex-wrap gap-1.5">
								{#each categories as category (category.id)}
									{@const selected = values.categoryIds.includes(category.id)}
									<button
										type="button"
										aria-pressed={selected}
										class={[
											'inline-flex items-center gap-1 rounded-full border px-2.5 py-1 text-xs font-medium transition-colors',
											selected
												? 'border-brand-200 bg-brand-200 text-brand-900'
												: 'border-zinc-300 bg-white text-zinc-700 hover:border-brand-500 hover:text-brand-700'
										]}
										onclick={() => toggleCategory(category.id)}
									>
										{#if selected}<Icon name="check" size={12} />{/if}
										{localized(category.name)}
									</button>
								{/each}
							</div>
						{/if}
						{#if errors.categoryIds}
							<p class="mt-2 text-xs text-red-600">{errors.categoryIds}</p>
						{/if}
					</fieldset>

					<label class="flex cursor-pointer items-start justify-between gap-3">
						<span>
							<span class="flex items-center gap-1.5 text-sm font-medium text-zinc-900">
								<Icon name="star" size={14} filled={values.featured} class="text-amber-400" />
								{m.pf_featured()}
							</span>
						</span>
						<input
							type="checkbox"
							role="switch"
							class="peer sr-only"
							bind:checked={values.featured}
						/>
						<span
							class="relative mt-0.5 h-5 w-9 shrink-0 rounded-full bg-zinc-200 transition-colors peer-checked:bg-brand-400 peer-focus-visible:ring-2 peer-focus-visible:ring-brand-600/40 after:absolute after:top-0.5 after:left-0.5 after:size-4 after:rounded-full after:bg-white after:shadow-sm after:transition-transform peer-checked:after:translate-x-4"
							aria-hidden="true"
						></span>
					</label>
				</div>
			</section>
		</fieldset>
	</div>

	<!-- Contextual save bar: shown while there is something to save. -->
	{#if !readOnly && (dirty || !product)}
		<div
			class="fixed inset-x-0 bottom-0 z-30 animate-slide-down border-t border-zinc-200 bg-white/95 shadow-[0_-4px_16px_-8px_rgb(15_23_42/0.15)] backdrop-blur lg:left-60"
		>
			<div class="mx-auto flex max-w-7xl items-center gap-3 px-4 py-3 sm:px-6 lg:px-8">
				<span
					class="mr-auto flex min-w-0 items-center gap-2 text-sm font-medium text-zinc-700"
					role="status"
				>
					{#if dirty}
						<span class="size-2 shrink-0 rounded-full bg-amber-400" aria-hidden="true"></span>
						<span class="truncate">{m.pf_unsaved_badge()}</span>
					{/if}
				</span>
				{#if dirty}
					<button type="button" class="btn-secondary" disabled={saving} onclick={discard}
						>{m.pf_discard()}</button
					>
				{/if}
				<button type="submit" class="btn-primary min-w-28" disabled={saving}>
					{#if saving}
						<span
							class="size-4 animate-spin rounded-full border-2 border-brand-900/20 border-t-brand-900"
							aria-hidden="true"
						></span>
						{m.common_saving()}
					{:else}
						{product ? m.common_save_changes() : m.pf_create()}
					{/if}
				</button>
			</div>
		</div>
	{/if}
</form>
