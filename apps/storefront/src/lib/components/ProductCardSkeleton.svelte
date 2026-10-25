<script lang="ts">
	import Skeleton from './Skeleton.svelte';
</script>

<!-- Same shape as ProductCard: image on the shelf, then text. -->
<div aria-hidden="true">
	<Skeleton class="aspect-square w-full rounded-t-md rounded-b-none" />
	<div class="shelf opacity-40"></div>
	<div class="space-y-2.5 pt-4">
		<Skeleton class="h-3.5 w-1/3" />
		<Skeleton class="h-4 w-3/4" />
		<Skeleton class="h-4 w-1/4" />
	</div>
</div>
