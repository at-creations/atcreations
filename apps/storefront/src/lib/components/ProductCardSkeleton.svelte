<script lang="ts">
	import Skeleton from './Skeleton.svelte';
</script>

<div class="overflow-hidden card" aria-hidden="true">
	<Skeleton class="aspect-[4/3] w-full rounded-none" />
	<div class="space-y-3 p-5">
		<Skeleton class="h-4 w-1/3" />
		<Skeleton class="h-5 w-3/4" />
		<Skeleton class="h-4 w-full" />
		<Skeleton class="h-4 w-1/4" />
	</div>
</div>
