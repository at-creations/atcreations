<script lang="ts">
	import { m } from '$lib/paraglide/messages.js';
	import { dismissToast, toasts } from '$lib/state/toast.svelte';
	import Icon from './Icon.svelte';

	const styles = {
		success: 'border-emerald-200 bg-emerald-50 text-emerald-900',
		error: 'border-red-200 bg-red-50 text-red-900',
		info: 'border-sky-200 bg-sky-50 text-sky-900'
	} as const;
	const icons = { success: 'check', error: 'alert', info: 'info' } as const;
</script>

<section
	aria-label={m.common_notifications()}
	class="pointer-events-none fixed inset-x-0 bottom-0 z-[60] flex flex-col items-center gap-2 p-4 sm:items-end"
>
	{#each toasts as t (t.id)}
		<div
			role={t.kind === 'error' ? 'alert' : 'status'}
			aria-live={t.kind === 'error' ? 'assertive' : 'polite'}
			class={[
				'pointer-events-auto flex w-full max-w-sm items-start gap-3 rounded-lg border p-3 shadow-lg',
				styles[t.kind]
			]}
		>
			<Icon name={icons[t.kind]} class="mt-0.5" />
			<div class="min-w-0 flex-1 text-sm">
				<p class="font-medium break-words">{t.message}</p>
				{#if t.detail}
					<p class="mt-0.5 break-words opacity-80">{t.detail}</p>
				{/if}
			</div>
			<button
				type="button"
				class="rounded p-0.5 opacity-70 hover:opacity-100"
				aria-label={m.common_dismiss()}
				onclick={() => dismissToast(t.id)}
			>
				<Icon name="x" size={16} />
			</button>
		</div>
	{/each}
</section>
