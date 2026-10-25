<script lang="ts">
	import type { BusinessHours } from '@atcreations/api-types';
	import { m } from '$lib/paraglide/messages';
	import { formatTime } from '$lib/format';
	import { groupBusinessHours, type DayKey } from '$lib/hours';
	import type { Locale } from '$lib/i18n';

	interface Props {
		hours: BusinessHours[];
		locale: Locale;
		/** Two columns from the sm breakpoint (wide panels), or always one (narrow columns). */
		columns?: 1 | 2;
	}

	let { hours, locale, columns = 2 }: Props = $props();

	const DAY_NAMES: Record<DayKey, () => string> = {
		mon: m.day_mon,
		tue: m.day_tue,
		wed: m.day_wed,
		thu: m.day_thu,
		fri: m.day_fri,
		sat: m.day_sat,
		sun: m.day_sun
	};

	const groups = $derived(groupBusinessHours(hours));

	function dayRange(from: DayKey, to: DayKey): string {
		return from === to ? DAY_NAMES[from]() : `${DAY_NAMES[from]()} – ${DAY_NAMES[to]()}`;
	}
</script>

{#if groups.length === 0}
	<p>{m.contact_to_be_updated()}</p>
{:else}
	<dl class={['grid grid-cols-1 gap-x-6 gap-y-3', columns === 2 && 'sm:grid-cols-2']}>
		{#each groups as group (group.from)}
			<div>
				<dt class="font-medium text-ink">
					{dayRange(group.from, group.to)}
				</dt>
				<dd>
					{#if group.closed || !group.open || !group.close}
						{m.contact_closed()}
					{:else}
						{formatTime(group.open, locale)} – {formatTime(group.close, locale)}
					{/if}
				</dd>
			</div>
		{/each}
	</dl>
{/if}
