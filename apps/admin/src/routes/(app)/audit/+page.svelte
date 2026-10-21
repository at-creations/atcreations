<script lang="ts">
	import type { AuditEntry, PageMeta } from '@atcreations/api-types';
	import { untrack } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api, unwrap } from '$lib/api';
	import LoadState from '$lib/components/LoadState.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Pagination from '$lib/components/Pagination.svelte';
	import RequirePermission from '$lib/components/RequirePermission.svelte';
	import { formatDateTime } from '$lib/i18n';
	import { m } from '$lib/paraglide/messages.js';
	import { getAdminSession } from '$lib/state/session.svelte';
	import { readInt, readString, withParams } from '$lib/utils/query';

	const session = getAdminSession();
	const uid = $props.id();
	const PAGE_SIZES = [25, 50, 100] as const;
	const RESOURCES: { value: string; label: () => string }[] = [
		{ value: 'product', label: m.audit_res_product },
		{ value: 'category', label: m.audit_res_category },
		{ value: 'order', label: m.audit_res_order },
		{ value: 'staff', label: m.audit_res_staff },
		{ value: 'invitation', label: m.audit_res_invitation },
		{ value: 'store', label: m.audit_res_store }
	];

	const filters = $derived.by(() => {
		const p = page.url.searchParams;
		return {
			page: readInt(p, 'page', 1),
			pageSize: readInt(p, 'pageSize', 25, PAGE_SIZES),
			resource: readString(p, 'resource', 40).replace(/[^a-z_.-]/gi, '')
		};
	});

	let entries = $state<AuditEntry[]>([]);
	let meta = $state<PageMeta | null>(null);
	let loading = $state(true);
	let loadError = $state<unknown>(null);
	let seq = 0;

	async function load(f: typeof filters) {
		const current = ++seq;
		loading = true;
		loadError = null;
		try {
			const res = await unwrap(
				api().GET('/v1/admin/audit', {
					params: {
						query: { page: f.page, pageSize: f.pageSize, resource: f.resource || undefined }
					}
				})
			);
			if (current !== seq) return;
			entries = res.data;
			meta = res.meta;
		} catch (error) {
			if (current === seq) loadError = error;
		} finally {
			if (current === seq) loading = false;
		}
	}

	$effect(() => {
		const f = filters;
		if (!session.can('audit:read')) return;
		untrack(() => load(f));
	});

	function navigate(changes: Record<string, string | number | null | undefined>) {
		// Same page, only the query string changes (built from page.url).
		// eslint-disable-next-line svelte/no-navigation-without-resolve
		goto(withParams(page.url, changes), { keepFocus: true, noScroll: true });
	}

	const knownResource = $derived(RESOURCES.some((r) => r.value === filters.resource));
</script>

<RequirePermission permission="audit:read">
	<PageHeader title={m.audit_title()} subtitle={m.audit_subtitle()} />

	<div class="mb-4 flex flex-wrap items-center gap-3 card p-4">
		<label for="{uid}-resource" class="text-sm font-medium text-slate-700"
			>{m.audit_filter_resource()}</label
		>
		<select
			id="{uid}-resource"
			class="input w-auto"
			value={filters.resource}
			onchange={(e) => navigate({ resource: e.currentTarget.value })}
		>
			<option value="">{m.audit_all_resources()}</option>
			{#each RESOURCES as r (r.value)}
				<option value={r.value}>{r.label()}</option>
			{/each}
			{#if filters.resource && !knownResource}
				<option value={filters.resource}>{filters.resource}</option>
			{/if}
		</select>
	</div>

	<section class="overflow-hidden card" aria-busy={loading}>
		{#if loadError}
			<LoadState error={loadError} fallback={m.audit_load_failed()} onretry={() => load(filters)} />
		{:else if loading && entries.length === 0}
			<LoadState loading />
		{:else if entries.length === 0}
			<p class="p-10 text-center text-slate-500">{m.audit_none()}</p>
		{:else}
			<div class="overflow-x-auto">
				<table class={['w-full min-w-[720px] text-left text-sm', loading && 'opacity-60']}>
					<thead class="bg-slate-50 text-xs font-semibold tracking-wide text-slate-500 uppercase">
						<tr>
							<th scope="col" class="px-4 py-3">{m.audit_col_time()}</th>
							<th scope="col" class="px-4 py-3">{m.audit_col_actor()}</th>
							<th scope="col" class="px-4 py-3">{m.audit_col_action()}</th>
							<th scope="col" class="px-4 py-3">{m.audit_col_resource()}</th>
							<th scope="col" class="px-4 py-3">{m.audit_col_resource_id()}</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-slate-100">
						{#each entries as entry (entry.id)}
							<tr class="hover:bg-slate-50">
								<td class="px-4 py-3 whitespace-nowrap text-slate-600">
									<time datetime={entry.at}>{formatDateTime(entry.at)}</time>
								</td>
								<td class="px-4 py-3">
									<span class="block">{entry.actorEmail || '—'}</span>
									<span class="block font-mono text-xs text-slate-400">{entry.actorId}</span>
								</td>
								<td class="px-4 py-3 font-mono text-xs">{entry.action}</td>
								<td class="px-4 py-3">{entry.resource}</td>
								<td class="px-4 py-3 font-mono text-xs break-all text-slate-600"
									>{entry.resourceId ?? '—'}</td
								>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
		{#if meta && !loadError}
			<Pagination
				{meta}
				pageSizes={[...PAGE_SIZES]}
				onpage={(p) => navigate({ page: p })}
				onpagesize={(size) => navigate({ pageSize: size })}
			/>
		{/if}
	</section>
</RequirePermission>
