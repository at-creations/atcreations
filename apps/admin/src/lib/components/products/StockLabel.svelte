<script lang="ts" module>
	/** At or below this many units a product counts as running low. */
	export const LOW_STOCK = 5;
</script>

<script lang="ts">
	import { m } from '$lib/paraglide/messages.js';

	let { stock }: { stock: number } = $props();

	const tone = $derived(
		stock === 0 ? 'bg-rose-400' : stock <= LOW_STOCK ? 'bg-amber-400' : 'bg-emerald-400'
	);
</script>

<span
	class={[
		'inline-flex items-center gap-1.5 text-xs whitespace-nowrap tabular-nums',
		stock === 0 ? 'font-medium text-rose-700' : 'text-zinc-600'
	]}
>
	<span class={['size-1.5 rounded-full', tone]} aria-hidden="true"></span>
	{stock === 0 ? m.dash_out_of_stock() : m.dash_in_stock({ count: stock })}
</span>
