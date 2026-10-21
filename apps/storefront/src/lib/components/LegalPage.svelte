<script lang="ts">
	import type { StoreInfo } from '@atcreations/api-types';
	import { m } from '$lib/paraglide/messages';
	import SectionTitle from './SectionTitle.svelte';

	export interface LegalSection {
		title: string;
		content: string;
		items?: string[];
	}

	interface Props {
		title: string;
		sections: LegalSection[];
		contactTitle: string;
		contactContent: string;
		storeInfo: StoreInfo | null;
	}

	let { title, sections, contactTitle, contactContent, storeInfo }: Props = $props();

	const email = $derived(storeInfo?.email?.trim() || '');
	const phone = $derived(storeInfo?.phone?.trim() || '');
	const address = $derived(storeInfo?.address?.trim() || '');
</script>

<div class="container-page animate-fade-in pt-28 pb-20">
	<SectionTitle level={1} {title} />

	<div class="prose-legal mx-auto max-w-4xl card p-6 md:p-8 lg:p-10">
		<p class="text-gray-600">
			{m.legal_last_updated()}: {m.legal_last_updated_date()}
		</p>

		{#each sections as section (section.title)}
			<h2>{section.title}</h2>
			<p>{section.content}</p>
			{#if section.items}
				<ul>
					{#each section.items as item (item)}
						<li>{item}</li>
					{/each}
				</ul>
			{/if}
		{/each}

		<h2>{contactTitle}</h2>
		<p>{contactContent}</p>
		<address class="not-italic">
			<ul>
				{#if email}
					<li>
						{m.contact_email()}:
						<a href={`mailto:${email}`} class="text-blue-700 hover:underline">{email}</a>
					</li>
				{/if}
				{#if phone}<li>{m.contact_phone()}: {phone}</li>{/if}
				<li>{m.legal_contact_address()}: {address || m.contact_to_be_updated()}</li>
			</ul>
		</address>
	</div>
</div>
