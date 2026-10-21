<script lang="ts">
	interface Props {
		class?: string;
	}

	let { class: className = '' }: Props = $props();
</script>

<div class={['animate-pulse rounded-md bg-gray-200 motion-reduce:animate-none', className]}></div>
