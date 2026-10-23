<script lang="ts">
	import { createQuery, keepPreviousData } from '@tanstack/svelte-query';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import LoadState from '$lib/components/LoadState.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Pagination from '$lib/components/Pagination.svelte';
	import RequirePermission from '$lib/components/RequirePermission.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import { formatDateTime } from '$lib/i18n';
	import { m } from '$lib/paraglide/messages.js';
	import { auditListQuery } from '$lib/queries';
	import { getAdminSession } from '$lib/state/session.svelte';
	import { readInt, readString, withParams } from '$lib/utils/query';

	const session = getAdminSession();
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

	const auditQuery = createQuery(() => ({
		...auditListQuery({
			page: filters.page,
			pageSize: filters.pageSize,
			resource: filters.resource || undefined
		}),
		placeholderData: keepPreviousData,
		enabled: session.can('audit:read')
	}));
	const entries = $derived(auditQuery.data?.data ?? []);
	const meta = $derived(auditQuery.data?.meta ?? null);
	const showError = $derived(auditQuery.isError && !auditQuery.data);

	function navigate(changes: Record<string, string | number | null | undefined>) {
		// Same page, only the query string changes (built from page.url).
		// eslint-disable-next-line svelte/no-navigation-without-resolve
		goto(withParams(page.url, changes), { keepFocus: true, noScroll: true });
	}

	const knownResource = $derived(RESOURCES.some((r) => r.value === filters.resource));

	const resourceItems = $derived([
		{ value: '', label: m.audit_all_resources() },
		...RESOURCES.map((r) => ({ value: r.value, label: r.label() })),
		...(filters.resource && !knownResource
			? [{ value: filters.resource, label: filters.resource }]
			: [])
	]);
</script>

<RequirePermission permission="audit:read">
	<PageHeader title={m.audit_title()} />

	<div class="mb-4 flex flex-wrap items-center gap-3 card p-3">
		<span class="text-sm font-medium text-zinc-700">{m.audit_filter_resource()}</span>
		<Select
			label={m.audit_filter_resource()}
			class="w-auto min-w-44"
			value={filters.resource}
			items={resourceItems}
			onchange={(resource) => navigate({ resource })}
		/>
	</div>

	<section class="overflow-hidden card" aria-busy={auditQuery.isFetching}>
		{#if showError}
			<LoadState
				error={auditQuery.error}
				fallback={m.audit_load_failed()}
				onretry={() => auditQuery.refetch()}
			/>
		{:else if auditQuery.isPending}
			<LoadState loading />
		{:else if entries.length === 0}
			<p class="p-10 text-center text-zinc-500">{m.audit_none()}</p>
		{:else}
			<div class="relative overflow-x-auto">
				<table
					class={[
						'w-full min-w-[720px] text-left text-sm',
						auditQuery.isPlaceholderData && 'opacity-60'
					]}
				>
					<thead class="border-b border-zinc-200 bg-zinc-50/80">
						<tr>
							<th scope="col" class="th">{m.audit_col_time()}</th>
							<th scope="col" class="th">{m.audit_col_actor()}</th>
							<th scope="col" class="th">{m.audit_col_action()}</th>
							<th scope="col" class="th">{m.audit_col_resource()}</th>
							<th scope="col" class="th">{m.audit_col_resource_id()}</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-zinc-100">
						{#each entries as entry (entry.id)}
							<tr class="hover:bg-zinc-50">
								<td class="td whitespace-nowrap text-zinc-600">
									<time datetime={entry.at}>{formatDateTime(entry.at)}</time>
								</td>
								<td class="td">
									<span class="block">{entry.actorEmail || '—'}</span>
									<span class="block font-mono text-xs text-zinc-400">{entry.actorId}</span>
								</td>
								<td class="td font-mono text-xs">{entry.action}</td>
								<td class="td">{entry.resource}</td>
								<td class="td font-mono text-xs break-all text-zinc-600"
									>{entry.resourceId ?? '—'}</td
								>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
		{#if meta && !showError}
			<Pagination
				{meta}
				pageSizes={[...PAGE_SIZES]}
				onpage={(p) => navigate({ page: p })}
				onpagesize={(size) => navigate({ pageSize: size })}
			/>
		{/if}
	</section>
</RequirePermission>
