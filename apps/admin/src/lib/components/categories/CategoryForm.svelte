<script lang="ts">
	import type { AdminCategory, Category, CategoryInput } from '@atcreations/api-types';
	import { untrack, type Snippet } from 'svelte';
	import { beforeNavigate, goto } from '$app/navigation';
	import { fieldErrorMap } from '$lib/api';
	import Field from '$lib/components/Field.svelte';
	import LangTabs from '$lib/components/LangTabs.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import SaveBar from '$lib/components/ui/SaveBar.svelte';
	import { toastError } from '$lib/notify';
	import { m } from '$lib/paraglide/messages.js';
	import { createSaveCategoryMutation } from '$lib/queries';
	import { confirmAction } from '$lib/state/confirm.svelte';
	import { getAdminSession } from '$lib/state/session.svelte';
	import { toast } from '$lib/state/toast.svelte';
	import { isValidSlug, slugify } from '$lib/utils/slug';

	let {
		category,
		nextSortOrder = 0,
		onsaved,
		aside
	}: {
		/** Category to edit; omit to create a new one. */
		category?: AdminCategory;
		/** Display position for a new category (the end of the list). */
		nextSortOrder?: number;
		onsaved?: (category: Category, created: boolean) => void | Promise<void>;
		/** Right column (image, products). */
		aside?: Snippet;
	} = $props();

	const session = getAdminSession();
	const save = createSaveCategoryMutation();
	const readOnly = $derived(!session.can('categories:write'));
	const saving = $derived(save.isPending);

	interface Values {
		nameEn: string;
		nameVi: string;
		descriptionEn: string;
		descriptionVi: string;
		slug: string;
	}
	const toValues = (c?: Category): Values => ({
		nameEn: c?.name.en ?? '',
		nameVi: c?.name.vi ?? '',
		descriptionEn: c?.description.en ?? '',
		descriptionVi: c?.description.vi ?? '',
		slug: c?.slug ?? ''
	});

	// Captured once; the page re-mounts this component per category.
	const initial = untrack(() => toValues(category));
	let values = $state<Values>(structuredClone(initial));
	let snapshot = $state(JSON.stringify(initial));
	let slugTouched = $state(untrack(() => !!category));
	let tab = $state<'en' | 'vi'>('en');
	let errors = $state<Record<string, string>>({});
	let bypassGuard = false;
	const dirty = $derived(JSON.stringify(values) !== snapshot);

	function validate(): CategoryInput | null {
		const e: Record<string, string> = {};
		if (!values.nameEn.trim()) e['name.en'] = m.validation_required();
		if (values.slug && !isValidSlug(values.slug, 80)) e.slug = m.validation_invalid_slug();
		errors = e;
		if (Object.keys(e).length) {
			if (e['name.en']) tab = 'en';
			return null;
		}
		const vi = values.nameVi.trim();
		const dVi = values.descriptionVi.trim();
		return {
			...(values.slug ? { slug: values.slug } : {}),
			name: { en: values.nameEn.trim(), ...(vi ? { vi } : {}) },
			description: { en: values.descriptionEn.trim(), ...(dVi ? { vi: dVi } : {}) },
			// Order is managed by dragging in the list; keep the current position.
			sortOrder: category?.sortOrder ?? nextSortOrder
		};
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (readOnly || saving) return;
		const body = validate();
		if (!body) {
			toast.error(m.validation_fix_errors());
			return;
		}
		try {
			const saved = await save.mutateAsync({ id: category?.id, body });
			const next = toValues(saved);
			values = next;
			snapshot = JSON.stringify(next);
			slugTouched = true;
			toast.success(
				category ? m.cat_updated({ name: saved.name.en }) : m.cat_created({ name: saved.name.en })
			);
			bypassGuard = true;
			try {
				await onsaved?.(saved, !category);
			} finally {
				bypassGuard = false;
			}
		} catch (error) {
			errors = fieldErrorMap(error);
			if (errors['name.en'] || errors['description.en']) tab = 'en';
			else if (errors['name.vi'] || errors['description.vi']) tab = 'vi';
			toastError(error, m.cat_save_failed());
		}
	}

	function discard() {
		values = JSON.parse(snapshot) as Values;
		errors = {};
	}

	beforeNavigate((navigation) => {
		if (!dirty || bypassGuard || readOnly) return;
		if (navigation.type === 'leave') {
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
</script>

<form onsubmit={submit} novalidate class="pb-24">
	<div class="grid items-start gap-6 lg:grid-cols-[minmax(0,1fr)_20rem]">
		<fieldset disabled={readOnly || saving} class="min-w-0">
			<Card title={m.pf_basic()}>
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
										class="input h-10 text-base font-medium"
										maxlength="100"
										placeholder={m.cat_name_placeholder()}
										value={values.nameEn}
										oninput={(e) => {
											values.nameEn = e.currentTarget.value;
											if (!slugTouched) values.slug = slugify(values.nameEn, 80);
										}}
									/>
								{/snippet}
							</Field>
							<Field label={m.cat_description()} optional error={errors['description.en']}>
								{#snippet children(control)}
									<textarea
										{...control}
										class="input h-auto min-h-32 py-2 leading-relaxed"
										maxlength="2000"
										placeholder={m.cat_description_placeholder()}
										bind:value={values.descriptionEn}></textarea>
								{/snippet}
							</Field>
						{:else}
							<Field label={m.cat_name()} optional error={errors['name.vi']}>
								{#snippet children(control)}
									<input
										{...control}
										class="input h-10 text-base font-medium"
										lang="vi"
										maxlength="100"
										placeholder={m.cat_name_vi_placeholder()}
										bind:value={values.nameVi}
									/>
								{/snippet}
							</Field>
							<Field label={m.cat_description()} optional error={errors['description.vi']}>
								{#snippet children(control)}
									<textarea
										{...control}
										class="input h-auto min-h-32 py-2 leading-relaxed"
										lang="vi"
										maxlength="2000"
										placeholder={m.cat_description_vi_placeholder()}
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
								<div
									class="flex h-9 min-w-0 flex-1 items-center rounded-lg border border-zinc-300 bg-zinc-50 shadow-xs focus-within:border-brand-400 focus-within:ring-3 focus-within:ring-brand-200/70"
								>
									<span class="pl-3 font-mono text-xs text-zinc-400">/products?category=</span>
									<input
										{...control}
										class="h-full min-w-0 flex-1 rounded-r-lg bg-white px-1.5 font-mono text-xs focus:outline-none"
										maxlength="80"
										autocomplete="off"
										spellcheck="false"
										value={values.slug}
										oninput={(e) => {
											slugTouched = true;
											values.slug = e.currentTarget.value.toLowerCase();
										}}
									/>
								</div>
								<button
									type="button"
									class="btn-secondary shrink-0"
									onclick={() => (values.slug = slugify(values.nameEn, 80))}
								>
									{m.pf_slug_suggest()}
								</button>
							</div>
						{/snippet}
					</Field>
				</div>
			</Card>
		</fieldset>

		<div class="min-w-0 space-y-6">
			{@render aside?.()}
		</div>
	</div>

	{#if !readOnly && (dirty || !category)}
		<SaveBar
			{dirty}
			{saving}
			submitLabel={category ? m.common_save_changes() : m.cat_create()}
			ondiscard={discard}
		/>
	{/if}
</form>
