<script lang="ts">
	import type { Role } from '@atcreations/api-types';
	import { permissionLabel, roleLabel, roleSummary } from '$lib/i18n';
	import { permissionsForRole, ROLES } from '$lib/utils/permissions';
	import Icon from '../Icon.svelte';

	let { highlight }: { highlight?: Role } = $props();
</script>

<ol class="space-y-2">
	{#each ROLES as role (role)}
		<li
			class={[
				'rounded-lg border p-3 text-sm',
				highlight === role ? 'border-brand-300 bg-brand-50' : 'border-zinc-200'
			]}
		>
			<p class="font-semibold">{roleLabel(role)}</p>
			<p class="text-xs text-zinc-600">{roleSummary(role)}</p>
			{#if highlight === role}
				<ul class="mt-2 grid gap-1 sm:grid-cols-2">
					{#each permissionsForRole(role) as permission (permission)}
						<li class="flex items-center gap-1 text-xs text-zinc-700">
							<Icon name="check" size={12} class="text-emerald-600" />{permissionLabel(permission)}
						</li>
					{/each}
				</ul>
			{/if}
		</li>
	{/each}
</ol>
