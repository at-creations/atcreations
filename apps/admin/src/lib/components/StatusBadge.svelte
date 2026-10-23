<script lang="ts">
	import type { OrderStatus, ProductStatus } from '@atcreations/api-types';
	import { orderStatusLabel, productStatusLabel, roleLabel } from '$lib/i18n';
	import type { StaffRole } from '$lib/utils/permissions';
	import Badge, { type BadgeTone } from './Badge.svelte';

	type Props =
		| { kind: 'product'; value: ProductStatus }
		| { kind: 'order'; value: OrderStatus }
		| { kind: 'role'; value: StaffRole };

	let props: Props = $props();

	const productTones: Record<ProductStatus, BadgeTone> = {
		draft: 'amber',
		published: 'green',
		archived: 'gray'
	};
	const orderTones: Record<OrderStatus, BadgeTone> = {
		pending: 'amber',
		confirmed: 'blue',
		processing: 'indigo',
		shipped: 'sky',
		delivered: 'green',
		completed: 'green',
		cancelled: 'red',
		refunded: 'purple'
	};
	const roleTones: Record<StaffRole, BadgeTone> = {
		admin: 'purple',
		manager: 'blue',
		staff: 'green',
		none: 'gray'
	};

	const view = $derived.by(() => {
		switch (props.kind) {
			case 'product':
				return { tone: productTones[props.value], label: productStatusLabel(props.value) };
			case 'order':
				return { tone: orderTones[props.value], label: orderStatusLabel(props.value) };
			default:
				return { tone: roleTones[props.value], label: roleLabel(props.value) };
		}
	});
</script>

<Badge tone={view.tone} dot>{view.label}</Badge>
