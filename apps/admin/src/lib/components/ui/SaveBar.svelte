<script lang="ts">
	import { m } from '$lib/paraglide/messages.js';

	let {
		dirty,
		saving,
		submitLabel,
		ondiscard
	}: {
		dirty: boolean;
		saving: boolean;
		submitLabel: string;
		/** Shown only while there are changes. */
		ondiscard?: () => void;
	} = $props();
</script>

<!-- Fixed bar along the bottom of the content area; render it inside the <form>. -->
<div
	class="fixed inset-x-0 bottom-0 z-30 animate-slide-down border-t border-zinc-200 bg-white/90 backdrop-blur lg:left-60"
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
		{#if dirty && ondiscard}
			<button type="button" class="btn-secondary" disabled={saving} onclick={ondiscard}
				>{m.pf_discard()}</button
			>
		{/if}
		<button type="submit" class="btn-primary min-w-28" disabled={saving}>
			{#if saving}
				<span
					class="size-4 animate-spin rounded-full border-2 border-current/25 border-t-current"
					aria-hidden="true"
				></span>
				{m.common_saving()}
			{:else}
				{submitLabel}
			{/if}
		</button>
	</div>
</div>
