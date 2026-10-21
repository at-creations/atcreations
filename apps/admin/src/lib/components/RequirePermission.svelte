<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { Permission } from '@atcreations/api-types';
	import { getAdminSession } from '$lib/state/session.svelte';
	import Forbidden from './Forbidden.svelte';

	let { permission, children }: { permission: Permission; children: Snippet } = $props();
	const session = getAdminSession();
</script>

{#if session.can(permission)}
	{@render children()}
{:else}
	<Forbidden />
{/if}
