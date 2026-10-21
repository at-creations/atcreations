<script lang="ts" module>
	// Minimal stroke icon set (paths adapted from Lucide, ISC licence).
	const ICONS = {
		dashboard: ['M3 3h7v9H3z', 'M14 3h7v5h-7z', 'M14 12h7v9h-7z', 'M3 16h7v5H3z'],
		box: [
			'M21 8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16Z',
			'm3.3 7 8.7 5 8.7-5',
			'M12 22V12'
		],
		tag: [
			'M12.586 2.586A2 2 0 0 0 11.172 2H4a2 2 0 0 0-2 2v7.172a2 2 0 0 0 .586 1.414l8.704 8.704a2.426 2.426 0 0 0 3.42 0l6.58-6.58a2.426 2.426 0 0 0 0-3.42z',
			'M7.5 7.5h.01'
		],
		cart: [
			'M8 21a1 1 0 1 0 0-2 1 1 0 0 0 0 2z',
			'M19 21a1 1 0 1 0 0-2 1 1 0 0 0 0 2z',
			'M2.05 2.05h2l2.66 12.42a2 2 0 0 0 2 1.58h9.78a2 2 0 0 0 1.95-1.57l1.65-7.43H5.12'
		],
		users: [
			'M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2',
			'M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8z',
			'M22 21v-2a4 4 0 0 0-3-3.87',
			'M16 3.13a4 4 0 0 1 0 7.75'
		],
		store: [
			'M3 9l1.5-5h15L21 9',
			'M3 9h18v2a3 3 0 0 1-6 0 3 3 0 0 1-6 0 3 3 0 0 1-6 0z',
			'M5 13v8h14v-8',
			'M10 21v-5h4v5'
		],
		history: ['M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8', 'M3 3v5h5', 'M12 7v5l4 2'],
		menu: ['M4 6h16', 'M4 12h16', 'M4 18h16'],
		x: ['M18 6 6 18', 'm6 6 12 12'],
		star: [
			'M11.525 2.295a.53.53 0 0 1 .95 0l2.31 4.679a2.123 2.123 0 0 0 1.595 1.16l5.166.756a.53.53 0 0 1 .294.904l-3.736 3.638a2.123 2.123 0 0 0-.611 1.878l.882 5.14a.53.53 0 0 1-.771.56l-4.618-2.428a2.122 2.122 0 0 0-1.973 0L6.396 21.01a.53.53 0 0 1-.77-.56l.881-5.139a2.122 2.122 0 0 0-.611-1.879L2.16 9.795a.53.53 0 0 1 .294-.906l5.165-.755a2.122 2.122 0 0 0 1.597-1.16z'
		],
		plus: ['M5 12h14', 'M12 5v14'],
		edit: ['M12 20h9', 'M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4Z'],
		trash: [
			'M3 6h18',
			'M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6',
			'M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2'
		],
		external: [
			'M15 3h6v6',
			'M10 14 21 3',
			'M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6'
		],
		up: ['m18 15-6-6-6 6'],
		down: ['m6 9 6 6 6-6'],
		left: ['m15 18-6-6 6-6'],
		right: ['m9 18 6-6-6-6'],
		search: ['M11 19a8 8 0 1 0 0-16 8 8 0 0 0 0 16z', 'm21 21-4.3-4.3'],
		check: ['M20 6 9 17l-5-5'],
		alert: [
			'm21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3',
			'M12 9v4',
			'M12 17h.01'
		],
		info: ['M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20z', 'M12 16v-4', 'M12 8h.01'],
		image: [
			'M19 3H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V5a2 2 0 0 0-2-2z',
			'M9 11a2 2 0 1 0 0-4 2 2 0 0 0 0 4z',
			'm21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21'
		],
		upload: ['M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4', 'm17 8-5-5-5 5', 'M12 3v12'],
		grip: ['M9 6h.01', 'M9 12h.01', 'M9 18h.01', 'M15 6h.01', 'M15 12h.01', 'M15 18h.01'],
		ban: ['M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20z', 'm4.9 4.9 14.2 14.2'],
		shield: [
			'M20 13c0 5-3.5 7.5-7.66 8.95a1 1 0 0 1-.67-.01C7.5 20.5 4 18 4 13V6a1 1 0 0 1 1-1c2 0 4.5-1.2 6.24-2.72a1.17 1.17 0 0 1 1.52 0C14.51 3.81 17 5 19 5a1 1 0 0 1 1 1z'
		],
		mail: [
			'M4 4h16a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2z',
			'm22 6-10 7L2 6'
		],
		globe: [
			'M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20z',
			'M2 12h20',
			'M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z'
		]
	} as const;

	export type IconName = keyof typeof ICONS;
</script>

<script lang="ts">
	let {
		name,
		size = 18,
		class: className = '',
		filled = false,
		label
	}: { name: IconName; size?: number; class?: string; filled?: boolean; label?: string } = $props();
</script>

<svg
	xmlns="http://www.w3.org/2000/svg"
	width={size}
	height={size}
	viewBox="0 0 24 24"
	fill={filled ? 'currentColor' : 'none'}
	stroke="currentColor"
	stroke-width="2"
	stroke-linecap="round"
	stroke-linejoin="round"
	class={['shrink-0', className]}
	role={label ? 'img' : undefined}
	aria-label={label}
	aria-hidden={label ? undefined : 'true'}
>
	{#each ICONS[name] as d, i (i)}
		<path {d} />
	{/each}
</svg>
