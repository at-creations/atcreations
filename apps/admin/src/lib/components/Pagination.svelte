<script lang="ts">
	import type { PageMeta } from '@atcreations/api-types';
	import { m } from '$lib/paraglide/messages.js';
	import Icon from './Icon.svelte';
	import Select from './ui/Select.svelte';

	let {
		meta,
		pageSizes = [10, 25, 50],
		onpage,
		onpagesize
	}: {
		meta: PageMeta;
		pageSizes?: number[];
		onpage: (page: number) => void;
		onpagesize?: (size: number) => void;
	} = $props();

	const start = $derived(meta.total === 0 ? 0 : (meta.page - 1) * meta.pageSize + 1);
	const end = $derived(Math.min(meta.page * meta.pageSize, meta.total));
	const totalPages = $derived(Math.max(1, meta.totalPages));
</script>

<nav
	aria-label={m.pagination_label()}
	class="flex flex-col items-center justify-between gap-3 border-t border-zinc-200 px-4 py-3 text-sm text-zinc-600 sm:flex-row"
>
	<p>{m.pagination_showing({ start, end, total: meta.total })}</p>
	<div class="flex items-center gap-3">
		{#if onpagesize}
			<div class="flex items-center gap-2">
				<span>{m.pagination_show()}</span>
				<Select
					size="sm"
					class="w-16"
					label={m.pagination_per_page()}
					value={String(meta.pageSize)}
					items={pageSizes.map((n) => ({ value: String(n), label: String(n) }))}
					onchange={(v) => onpagesize?.(Number(v))}
				/>
			</div>
		{/if}
		<button
			type="button"
			class="btn-secondary btn-sm"
			aria-label={m.pagination_prev()}
			disabled={meta.page <= 1}
			onclick={() => onpage(meta.page - 1)}
		>
			<Icon name="left" size={16} />
		</button>
		<span aria-live="polite">{m.pagination_page({ current: meta.page, total: totalPages })}</span>
		<button
			type="button"
			class="btn-secondary btn-sm"
			aria-label={m.pagination_next()}
			disabled={meta.page >= totalPages}
			onclick={() => onpage(meta.page + 1)}
		>
			<Icon name="right" size={16} />
		</button>
	</div>
</nav>
