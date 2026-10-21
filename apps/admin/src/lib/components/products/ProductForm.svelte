<script lang="ts">
	import type { Category, Product, ProductInput, ProductStatus } from '@atcreations/api-types';
	import { untrack } from 'svelte';
	import { beforeNavigate, goto } from '$app/navigation';
	import { api, fieldErrorMap, unwrap } from '$lib/api';
	import Field from '$lib/components/Field.svelte';
	import Icon from '$lib/components/Icon.svelte';
	import LangTabs from '$lib/components/LangTabs.svelte';
	import { localized, priceErrorMessage, productStatusLabel } from '$lib/i18n';
	import { toastError } from '$lib/notify';
	import { m } from '$lib/paraglide/messages.js';
	import { confirmAction } from '$lib/state/confirm.svelte';
	import { getAdminSession } from '$lib/state/session.svelte';
	import { toast } from '$lib/state/toast.svelte';
	import { centsToDollarInput, formatCAD, parseDollars } from '$lib/utils/money';
	import { isValidSlug, slugify } from '$lib/utils/slug';

	let {
		product,
		categories,
		onsaved
	}: {
		product?: Product;
		categories: Category[];
		onsaved: (product: Product, created: boolean) => void | Promise<void>;
	} = $props();

	const session = getAdminSession();
	const readOnly = $derived(!session.can('products:write'));
	const uid = $props.id();
	const STATUSES: readonly ProductStatus[] = ['draft', 'published', 'archived'];

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
	let saving = $state(false);
	let bypassGuard = false;

	const dirty = $derived(JSON.stringify(values) !== snapshot);
	const parsedPrice = $derived(parseDollars(values.price));
	const finalPreview = $derived.by(() => {
		if (!parsedPrice.ok || parsedPrice.cents === null) return null;
		const d = intInRange(values.discount, 0, 90) ?? 0;
		// Mirrors the API: rounded half-up.
		return Math.floor((parsedPrice.cents * (100 - d)) / 100 + 0.5);
	});

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
			...(product ? { version: product.version } : {})
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
		saving = true;
		try {
			const res = product
				? await unwrap(
						api().PUT('/v1/admin/products/{id}', { params: { path: { id: product.id } }, body })
					)
				: await unwrap(api().POST('/v1/admin/products', { body }));
			const saved = res.data;
			const next = toValues(saved);
			values = next;
			snapshot = JSON.stringify(next);
			slugTouched = true;
			toast.success(product ? m.pf_updated() : m.pf_created());
			bypassGuard = true;
			await onsaved(saved, !product);
			bypassGuard = false;
		} catch (error) {
			errors = fieldErrorMap(error);
			if (errors['name.en'] || errors['description.en']) tab = 'en';
			else if (errors['name.vi'] || errors['description.vi']) tab = 'vi';
			toastError(error, m.pf_save_failed());
		} finally {
			saving = false;
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

<form onsubmit={submit} novalidate class="space-y-6">
	{#if readOnly}
		<p class="rounded-lg bg-amber-50 p-3 text-sm text-amber-900">{m.pf_read_only()}</p>
	{/if}
	<fieldset disabled={readOnly || saving} class="space-y-6">
		<section class="card p-5">
			<h2 class="mb-4 text-lg font-semibold">{m.pf_basic()}</h2>
			{#if product}
				<p class="mb-4 text-xs text-slate-500">
					{m.pf_product_id()}: <span class="font-mono select-all">{product.id}</span>
				</p>
			{/if}
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
									class="input min-h-32"
									maxlength="5000"
									placeholder={m.pf_description_placeholder()}
									bind:value={values.descriptionEn}></textarea>
							{/snippet}
						</Field>
					{:else}
						<p class="text-xs text-slate-500">{m.vietnamese_note()}</p>
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
									class="input min-h-32"
									lang="vi"
									maxlength="5000"
									placeholder={m.pf_description_vi_placeholder()}
									bind:value={values.descriptionVi}></textarea>
							{/snippet}
						</Field>
					{/if}
				{/snippet}
			</LangTabs>

			<div class="mt-4 grid gap-4 md:grid-cols-2">
				<Field label={m.pf_slug()} hint={m.pf_slug_hint()} error={errors.slug}>
					{#snippet children(control)}
						<div class="flex gap-2">
							<input
								{...control}
								class="input font-mono"
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
				<Field label={m.pf_status()} error={errors.status}>
					{#snippet children(control)}
						<select {...control} class="input" bind:value={values.status}>
							{#each STATUSES as status (status)}
								<option value={status}>{productStatusLabel(status)}</option>
							{/each}
						</select>
					{/snippet}
				</Field>
			</div>
		</section>

		<section class="card p-5">
			<h2 class="mb-4 text-lg font-semibold">{m.pf_pricing()}</h2>
			<div class="grid gap-4 md:grid-cols-3">
				<Field label={m.pf_price()} hint={m.pf_price_hint()} error={errors.priceCents}>
					{#snippet children(control)}
						<input
							{...control}
							class="input"
							inputmode="decimal"
							placeholder="0.00"
							bind:value={values.price}
						/>
					{/snippet}
				</Field>
				<Field label={m.pf_discount()} hint={m.pf_discount_hint()} error={errors.discountPercent}>
					{#snippet children(control)}
						<input
							{...control}
							class="input"
							type="number"
							min="0"
							max="90"
							step="1"
							inputmode="numeric"
							bind:value={values.discount}
						/>
					{/snippet}
				</Field>
				<Field label={m.pf_stock()} error={errors.stock}>
					{#snippet children(control)}
						<input
							{...control}
							class="input"
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
			<p class="mt-2 text-sm text-slate-600" aria-live="polite">
				{finalPreview === null
					? m.products_contact_price()
					: m.pf_final_price({ price: formatCAD(finalPreview) })}
			</p>
			<label class="mt-4 inline-flex items-center gap-2 text-sm font-medium">
				<input
					type="checkbox"
					class="size-4 rounded border-slate-300"
					bind:checked={values.featured}
				/>
				<Icon name="star" size={16} filled={values.featured} class="text-amber-400" />
				{m.pf_featured()}
			</label>
		</section>

		<section class="card p-5">
			<fieldset>
				<legend class="mb-1 text-lg font-semibold">
					{m.pf_categories()}
					<span class="text-sm font-normal text-slate-500"
						>{m.pf_categories_selected({ count: values.categoryIds.length })}</span
					>
				</legend>
				<p id="{uid}-cat-help" class="mb-3 text-xs text-slate-500">{m.pf_categories_help()}</p>
				{#if categories.length === 0}
					<p class="text-sm text-slate-500">{m.pf_no_categories()}</p>
				{:else}
					<div class="flex flex-wrap gap-2" aria-describedby="{uid}-cat-help">
						{#each categories as category (category.id)}
							{@const selected = values.categoryIds.includes(category.id)}
							<button
								type="button"
								aria-pressed={selected}
								class={[
									'inline-flex items-center gap-1 rounded-full border px-3 py-1 text-sm transition',
									selected
										? 'border-brand-800 bg-brand-800 text-white'
										: 'border-slate-300 bg-white text-slate-700 hover:border-brand-600'
								]}
								onclick={() => toggleCategory(category.id)}
							>
								{#if selected}<Icon name="check" size={14} />{/if}
								{localized(category.name)}
							</button>
						{/each}
					</div>
				{/if}
				{#if errors.categoryIds}<p class="mt-2 text-xs text-red-600">{errors.categoryIds}</p>{/if}
			</fieldset>
		</section>
	</fieldset>

	{#if !readOnly}
		<div
			class="sticky bottom-0 z-10 -mx-4 flex items-center justify-end gap-3 border-t border-slate-200 bg-white/95 px-4 py-3 backdrop-blur sm:mx-0 sm:rounded-xl sm:border"
		>
			{#if dirty}
				<span class="mr-auto text-sm font-medium text-amber-700" role="status"
					>{m.pf_unsaved_badge()}</span
				>
			{/if}
			<button type="submit" class="btn-primary" disabled={saving}>
				{#if saving}
					<span class="size-4 animate-spin rounded-full border-2 border-white/40 border-t-white"
					></span>
					{m.common_saving()}
				{:else}
					{product ? m.pf_update() : m.pf_create()}
				{/if}
			</button>
		</div>
	{/if}
</form>
