<script lang="ts">
	import type { Category, CategoryInput } from '@atcreations/api-types';
	import { fieldErrorMap } from '$lib/api';
	import Field from '$lib/components/Field.svelte';
	import LangTabs from '$lib/components/LangTabs.svelte';
	import Modal from '$lib/components/Modal.svelte';
	import { toastError } from '$lib/notify';
	import { m } from '$lib/paraglide/messages.js';
	import { createSaveCategoryMutation } from '$lib/queries';
	import { toast } from '$lib/state/toast.svelte';
	import { isValidSlug, slugify } from '$lib/utils/slug';

	let {
		open = $bindable(false),
		category,
		nextSortOrder = 0
	}: {
		open?: boolean;
		/** Category to edit; omit to create a new one. */
		category?: Category | null;
		nextSortOrder?: number;
	} = $props();

	const uid = $props.id();

	let nameEn = $state('');
	let nameVi = $state('');
	let descriptionEn = $state('');
	let descriptionVi = $state('');
	let slug = $state('');
	let sortOrder = $state<number | null>(0);
	let slugTouched = $state(false);
	let tab = $state<'en' | 'vi'>('en');
	let errors = $state<Record<string, string>>({});
	const save = createSaveCategoryMutation();
	const saving = $derived(save.isPending);

	/** Reset the form whenever the dialog is opened. */
	export function reset() {
		nameEn = category?.name.en ?? '';
		nameVi = category?.name.vi ?? '';
		descriptionEn = category?.description.en ?? '';
		descriptionVi = category?.description.vi ?? '';
		slug = category?.slug ?? '';
		sortOrder = category?.sortOrder ?? nextSortOrder;
		slugTouched = !!category;
		tab = 'en';
		errors = {};
	}

	function validate(): CategoryInput | null {
		const e: Record<string, string> = {};
		if (!nameEn.trim()) e['name.en'] = m.validation_required();
		if (slug && !isValidSlug(slug, 80)) e.slug = m.validation_invalid_slug();
		const order = sortOrder ?? NaN;
		if (!Number.isInteger(order) || order < 0 || order > 10000)
			e.sortOrder = m.validation_int_range({ min: 0, max: 10000 });
		errors = e;
		if (Object.keys(e).length) {
			if (e['name.en']) tab = 'en';
			return null;
		}
		const vi = nameVi.trim();
		const dEn = descriptionEn.trim();
		const dVi = descriptionVi.trim();
		return {
			...(slug ? { slug } : {}),
			name: { en: nameEn.trim(), ...(vi ? { vi } : {}) },
			description: { en: dEn, ...(dVi ? { vi: dVi } : {}) },
			sortOrder: order
		};
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (saving) return;
		const body = validate();
		if (!body) return;
		try {
			const saved = await save.mutateAsync({ id: category?.id, body });
			toast.success(
				category ? m.cat_updated({ name: saved.name.en }) : m.cat_created({ name: saved.name.en })
			);
			open = false;
		} catch (error) {
			errors = fieldErrorMap(error);
			if (errors['name.en'] || errors['description.en']) tab = 'en';
			toastError(error, m.cat_save_failed());
		}
	}
</script>

<Modal
	bind:open
	busy={saving}
	size="lg"
	title={category ? m.cat_edit_title() : m.cat_create_title()}
>
	<form id="{uid}-form" onsubmit={submit} novalidate class="space-y-4">
		<LangTabs
			bind:active={tab}
			errors={{
				en: !!(errors['name.en'] || errors['description.en']),
				vi: !!(errors['name.vi'] || errors['description.vi'])
			}}
		>
			{#snippet children(lang)}
				{#if lang === 'en'}
					<Field label={m.cat_name()} required error={errors['name.en']}>
						{#snippet children(control)}
							<input
								{...control}
								class="input"
								maxlength="100"
								placeholder={m.cat_name_placeholder()}
								value={nameEn}
								oninput={(e) => {
									nameEn = e.currentTarget.value;
									if (!slugTouched) slug = slugify(nameEn, 80);
								}}
							/>
						{/snippet}
					</Field>
					<Field label={m.cat_description()} optional error={errors['description.en']}>
						{#snippet children(control)}
							<textarea
								{...control}
								class="input min-h-24"
								maxlength="2000"
								placeholder={m.cat_description_placeholder()}
								bind:value={descriptionEn}></textarea>
						{/snippet}
					</Field>
				{:else}
					<p class="text-xs text-slate-500">{m.vietnamese_note()}</p>
					<Field label={m.cat_name()} optional error={errors['name.vi']}>
						{#snippet children(control)}
							<input
								{...control}
								class="input"
								lang="vi"
								maxlength="100"
								placeholder={m.cat_name_vi_placeholder()}
								bind:value={nameVi}
							/>
						{/snippet}
					</Field>
					<Field label={m.cat_description()} optional error={errors['description.vi']}>
						{#snippet children(control)}
							<textarea
								{...control}
								class="input min-h-24"
								lang="vi"
								maxlength="2000"
								placeholder={m.cat_description_vi_placeholder()}
								bind:value={descriptionVi}></textarea>
						{/snippet}
					</Field>
				{/if}
			{/snippet}
		</LangTabs>
		<div class="grid gap-4 sm:grid-cols-2">
			<Field label={m.cat_slug()} hint={m.cat_slug_hint()} error={errors.slug}>
				{#snippet children(control)}
					<input
						{...control}
						class="input font-mono"
						maxlength="80"
						autocomplete="off"
						spellcheck="false"
						value={slug}
						oninput={(e) => {
							slugTouched = true;
							slug = e.currentTarget.value.toLowerCase();
						}}
					/>
				{/snippet}
			</Field>
			<Field label={m.cat_sort_order()} hint={m.cat_sort_hint()} error={errors.sortOrder}>
				{#snippet children(control)}
					<input
						{...control}
						class="input"
						type="number"
						min="0"
						max="10000"
						step="1"
						bind:value={sortOrder}
					/>
				{/snippet}
			</Field>
		</div>
	</form>
	{#snippet footer()}
		<button type="button" class="btn-secondary" disabled={saving} onclick={() => (open = false)}>
			{m.common_cancel()}
		</button>
		<button type="submit" form="{uid}-form" class="btn-primary" disabled={saving}>
			{saving ? m.common_saving() : category ? m.common_save_changes() : m.cat_create()}
		</button>
	{/snippet}
</Modal>
